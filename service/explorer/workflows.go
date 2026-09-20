package explorer

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/workflows"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"math"
	"slices"
)

// ItemMoveService 处理多文件/目录移动
type ItemMoveService struct {
	SrcDir string        `json:"src_dir" binding:"required,min=1,max=65535"`
	Src    ItemIDService `json:"src"`
	Dst    string        `json:"dst" binding:"required,min=1,max=65535"`
}

// ItemRenameService 处理多文件/目录重命名
type ItemRenameService struct {
	Src     ItemIDService `json:"src"`
	NewName string        `json:"new_name" binding:"required,min=1,max=255"`
}

// ItemService 处理多文件/目录相关服务
type ItemService struct {
	Items []uint `json:"items"`
	Dirs  []uint `json:"dirs"`
}

// ItemIDService 处理多文件/目录相关服务，字段值为HashID，可通过Raw()方法获取原始ID
type ItemIDService struct {
	Items      []string `json:"items"`
	Dirs       []string `json:"dirs"`
	Source     *ItemService
	Force      bool `json:"force"`
	UnlinkOnly bool `json:"unlink"`
}

// ItemDecompressService 文件解压缩任务服务
type ItemDecompressService struct {
	Src      string `json:"src"`
	Dst      string `json:"dst" binding:"required,min=1,max=65535"`
	Encoding string `json:"encoding"`
}

// ItemPropertyService 获取对象属性服务
type ItemPropertyService struct {
	ID        string `binding:"required"`
	TraceRoot bool   `form:"trace_root"`
	IsFolder  bool   `form:"is_folder"`
}

func init() {
	gob.Register(ItemIDService{})
}

type (
	DownloadWorkflowService struct {
		Src        []string `json:"src"`
		SrcFile    string   `json:"src_file"`
		Dst        string   `json:"dst" binding:"required"`
		FileName   string   `json:"file_name" binding:"omitempty,max=255"`
		Username   string   `json:"username" binding:"omitempty,max=255"`
		Password   string   `json:"password" binding:"omitempty,max=255"`
		Headers    []string `json:"headers" binding:"omitempty,max=32,dive,max=2048"`
		Provider   string   `json:"provider" binding:"omitempty,max=64"`
		TargetNode string   `json:"target_node" binding:"omitempty,max=64"`
	}
	CreateDownloadParamCtx struct{}
)

// resolveNodeSelection validates the caller's target_node pick against group
// constraints and returns the NodeSelection for task creation. An empty
// target means auto dispatch; the group's allowed pool always applies.
func resolveNodeSelection(c *gin.Context, dep dependency.Dep, target string, capability types.NodeCapability) (*workflows.NodeSelection, error) {
	user := inventory.UserFromContext(c)
	sel := &workflows.NodeSelection{AllowedNodes: inventory.EffectiveGroup(user).Settings.AllowedNodes}

	if target == "" {
		return sel, nil
	}

	if !inventory.EffectiveGroup(user).Settings.AllowSelectNode {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Group not allowed to select node", nil)
	}

	nodeID, err := dep.HashIDEncoder().Decode(target, hashid.NodeID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid target node", err)
	}

	if len(sel.AllowedNodes) > 0 && !slices.Contains(sel.AllowedNodes, nodeID) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Target node not allowed for this group", nil)
	}

	n, err := dep.NodeClient().GetNodeById(c, nodeID)
	if err != nil || n.Status != node.StatusActive ||
		n.Capabilities == nil || !n.Capabilities.Enabled(int(capability)) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Target node unavailable", err)
	}

	sel.TargetNodeID = nodeID
	return sel, nil
}

func (service *DownloadWorkflowService) CreateDownloadTask(c *gin.Context) ([]*TaskResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	if !inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionRemoteDownload)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Group not allowed to download files", nil)
	}

	// Src must be set
	if service.SrcFile == "" && len(service.Src) == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "No source files", nil)
	}

	// Only one of src and src_file can be set
	if service.SrcFile != "" && len(service.Src) > 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid source files", nil)
	}

	dst, err := fs.NewUriFromString(service.Dst)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid destination", err)
	}

	// Validate dst
	_, err = m.Get(c, dst, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityCreateFile))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid destination", err)
	}

	// 检查批量任务数量
	limit := inventory.EffectiveGroup(user).Settings.Aria2BatchSize
	if limit > 0 && len(service.Src) > limit {
		return nil, serializer.NewError(serializer.CodeBatchAria2Size, "", nil)
	}

	// Concurrent active-task quota for this user's group.
	if taskLimit := inventory.EffectiveGroup(user).Settings.Aria2TaskLimit; taskLimit > 0 {
		active, err := dep.TaskClient().List(c, &inventory.ListTaskArgs{
			PaginationArgs: &inventory.PaginationArgs{PageSize: 1},
			UserID:         user.ID,
			Types:          []string{queue.RemoteDownloadTaskType},
			Status:         []task.Status{task.StatusQueued, task.StatusProcessing, task.StatusSuspending},
		})
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to count download tasks", err)
		}
		if active.PaginationResults.TotalItems >= taskLimit {
			return nil, serializer.NewError(serializer.CodeBatchAria2Size,
				fmt.Sprintf("Concurrent download task limit reached (%d)", taskLimit), nil)
		}
	}

	// Validate src file
	if service.SrcFile != "" {
		src, err := fs.NewUriFromString(service.SrcFile)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid source file uri", err)
		}

		_, err = m.Get(c, src, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile))
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid source file", err)
		}
	}

	// Validate custom request headers: "Name: value" lines, no CRLF injection.
	for _, h := range service.Headers {
		name, _, ok := strings.Cut(h, ":")
		if !ok || strings.TrimSpace(name) == "" || strings.ContainsAny(h, "\r\n") {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid header", nil)
		}
	}

	// Distinguish BitTorrent sources (magnet links, .torrent files) from
	// plain URLs: they require a BT-capable provider, and src_file must
	// actually name a .torrent.
	if service.SrcFile != "" && !strings.HasSuffix(strings.ToLower(service.SrcFile), ".torrent") {
		return nil, serializer.NewError(serializer.CodeParamErr, "Source file must be a .torrent", nil)
	}
	torrentSrc := service.SrcFile != ""
	for _, s := range service.Src {
		if strings.HasPrefix(strings.ToLower(s), "magnet:") {
			torrentSrc = true
			break
		}
	}
	if torrentSrc {
		if service.Provider == string(types.DownloaderProviderYtDlp) {
			return nil, serializer.NewError(serializer.CodeParamErr, "Downloader does not support torrent sources", nil)
		}
		if service.Provider == "" {
			// Auto-pick a BitTorrent-capable provider, preferring qBittorrent.
			for _, p := range []types.DownloaderProvider{types.DownloaderProviderQBittorrent, types.DownloaderProviderAria2} {
				if providerNodeAvailable(c, dep, string(p)) != 0 {
					service.Provider = string(p)
					break
				}
			}
			if service.Provider == "" {
				return nil, serializer.NewError(serializer.CodeParamErr, "No BitTorrent-capable downloader node available", nil)
			}
		}
	}

	// Validate a requested downloader provider against what the node pool
	// actually offers.
	if service.Provider != "" && providerNodeAvailable(c, dep, service.Provider) == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid downloader provider", nil)
	}

	nodeSel, err := resolveNodeSelection(c, dep, service.TargetNode, types.NodeCapabilityRemoteDownload)
	if err != nil {
		return nil, err
	}

	// Custom file name only applies to single-source tasks; HTTP credentials
	// and headers only apply to plain HTTP(S) source URLs.
	taskOpts := &workflows.RemoteDownloadTaskOption{
		Provider:     service.Provider,
		HTTPUsername: service.Username,
		HTTPPassword: service.Password,
		HTTPHeaders:  service.Headers,
		NodeSel:      *nodeSel,
	}
	if len(service.Src) <= 1 {
		taskOpts.FileName = service.FileName
	}
	if service.SrcFile != "" {
		taskOpts.HTTPUsername = ""
		taskOpts.HTTPPassword = ""
		taskOpts.HTTPHeaders = nil
	}

	// batch creating tasks
	ae := serializer.NewAggregateError()
	tasks := make([]queue.Task, 0, len(service.Src))
	for _, src := range service.Src {
		if src == "" {
			continue
		}

		t, err := workflows.NewRemoteDownloadTask(c, src, service.SrcFile, service.Dst, taskOpts)
		if err != nil {
			ae.Add(src, err)
			continue
		}

		if err := dep.RemoteDownloadQueue(c).QueueTask(c, t); err != nil {
			ae.Add(src, err)
		}

		tasks = append(tasks, t)
	}

	if service.SrcFile != "" {
		t, err := workflows.NewRemoteDownloadTask(c, "", service.SrcFile, service.Dst, taskOpts)
		if err != nil {
			ae.Add(service.SrcFile, err)
		}

		if err := dep.RemoteDownloadQueue(c).QueueTask(c, t); err != nil {
			ae.Add(service.SrcFile, err)
		}

		tasks = append(tasks, t)
	}

	return lo.Map(tasks, func(item queue.Task, index int) *TaskResponse {
		return BuildTaskResponse(item, nil, hasher)
	}), ae.Aggregate()
}

type (
	ArchiveWorkflowService struct {
		Src        []string `json:"src" binding:"required"`
		Dst        string   `json:"dst" binding:"required"`
		Encoding   string   `json:"encoding"`
		Password   string   `json:"password"`
		FileMask   []string `json:"file_mask"`
		TargetNode string   `json:"target_node" binding:"omitempty,max=64"`
	}
	CreateArchiveParamCtx struct{}
)

func (service *ArchiveWorkflowService) CreateExtractTask(c *gin.Context) (*TaskResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	if !inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionArchiveTask)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Group not allowed to compress files", nil)
	}

	dst, err := fs.NewUriFromString(service.Dst)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid destination", err)
	}

	if len(service.Src) == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "No source files", nil)
	}

	// Validate destination
	if _, err := m.Get(c, dst, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityCreateFile)); err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid destination", err)
	}

	// When multiple sources form a single multi-volume archive set, extract
	// from the first volume and pass all volumes to the task.
	src := service.Src[0]
	var volumes []string
	if first, ok := workflows.FirstVolumeURI(service.Src); ok {
		src = first
		volumes = service.Src
	}

	nodeSel, err := resolveNodeSelection(c, dep, service.TargetNode, types.NodeCapabilityExtractArchive)
	if err != nil {
		return nil, err
	}

	// Create task
	t, err := workflows.NewExtractArchiveTask(c, src, service.Dst, service.Encoding, service.Password, service.FileMask, volumes, *nodeSel)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to create task", err)
	}

	if err := dep.IoIntenseQueue(c).QueueTask(c, t); err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to queue task", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventExtractArchive,
		activity.Extra(map[string]any{"src": service.Src, "dst": service.Dst}))
	return BuildTaskResponse(t, nil, hasher), nil
}

// CreateCompressTask Create task to create an archive file
func (service *ArchiveWorkflowService) CreateCompressTask(c *gin.Context) (*TaskResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	if !inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionArchiveTask)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Group not allowed to compress files", nil)
	}

	dst, err := fs.NewUriFromString(service.Dst)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid destination", err)
	}

	// Create a placeholder file then delete it to validate the destination
	session, err := m.PrepareUpload(c, &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:             dst,
			Size:            0,
			UploadSessionID: uuid.Must(uuid.NewV4()).String(),
			ExpireAt:        time.Now().Add(time.Second * 3600),
		},
	})
	if err != nil {
		return nil, err
	}
	m.OnUploadFailed(c, session)

	nodeSel, err := resolveNodeSelection(c, dep, service.TargetNode, types.NodeCapabilityCreateArchive)
	if err != nil {
		return nil, err
	}

	// Create task
	t, err := workflows.NewCreateArchiveTask(c, service.Src, service.Dst, *nodeSel)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to create task", err)
	}

	if err := dep.IoIntenseQueue(c).QueueTask(c, t); err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to queue task", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventCreateArchive,
		activity.Extra(map[string]any{"src": service.Src, "dst": service.Dst}))
	return BuildTaskResponse(t, nil, hasher), nil
}

type (
	ImportWorkflowService struct {
		Src              string `json:"src" binding:"required"`
		Dst              string `json:"dst" binding:"required"`
		ExtractMediaMeta bool   `json:"extract_media_meta"`
		UserID           string `json:"user_id" binding:"required"`
		Recursive        bool   `json:"recursive"`
		PolicyID         int    `json:"policy_id" binding:"required"`
	}
	CreateImportParamCtx struct{}
)

func (service *ImportWorkflowService) CreateImportTask(c *gin.Context) (*TaskResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	if !inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Only admin can import files", nil)
	}

	userId, err := hasher.Decode(service.UserID, hashid.UserID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid user id", err)
	}

	owner, err := dep.UserClient().GetLoginUserByID(c, userId)
	if err != nil || owner.ID == 0 {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user", err)
	}

	dst, err := fs.NewUriFromString(fs.NewMyUri(service.UserID))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid destination", err)
	}

	// Resolve policy name for the task summary; best-effort, the task itself
	// validates the policy at execution time.
	var policyName string
	if policy, err := dep.StoragePolicyClient().GetPolicyByID(c, service.PolicyID); err == nil {
		policyName = policy.Name
	}

	// Create task
	t, err := workflows.NewImportTask(c, owner, service.Src, service.Recursive, dst.Join(service.Dst).String(), service.PolicyID, policyName)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to create task", err)
	}

	if err := dep.IoIntenseQueue(c).QueueTask(c, t); err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to queue task", err)
	}

	return BuildTaskResponse(t, nil, hasher), nil
}

type (
	ListTaskService struct {
		PageSize      int    `form:"page_size" binding:"required,min=10,max=100"`
		Category      string `form:"category" binding:"required,eq=general|eq=downloading|eq=downloaded"`
		NextPageToken string `form:"next_page_token"`
	}
	ListTaskParamCtx struct{}
)

func (service *ListTaskService) ListTasks(c *gin.Context) (*TaskListResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	taskClient := dep.TaskClient()

	args := &inventory.ListTaskArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageToken:           service.NextPageToken,
			PageSize:            service.PageSize,
		},
		Types:         []string{queue.CreateArchiveTaskType, queue.ExtractArchiveTaskType, queue.RelocateTaskType, queue.ImportTaskType},
		UserID:        user.ID,
		ExcludeHidden: true,
	}

	if service.Category != "general" {
		args.Types = []string{queue.RemoteDownloadTaskType}
		if service.Category == "downloading" {
			args.PageSize = math.MaxInt
			args.Status = []task.Status{task.StatusSuspending, task.StatusProcessing, task.StatusQueued}
		} else if service.Category == "downloaded" {
			args.Status = []task.Status{task.StatusCanceled, task.StatusError, task.StatusCompleted}
		}
	}

	// Get tasks
	res, err := taskClient.List(c, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to query tasks", err)
	}

	tasks := make([]queue.Task, 0, len(res.Tasks))
	nodeMap := make(map[int]*ent.Node)
	for _, t := range res.Tasks {
		task, err := queue.NewTaskFromModel(t)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to parse task", err)
		}

		summary := task.Summarize(hasher)
		if summary != nil && summary.NodeID > 0 {
			if _, ok := nodeMap[summary.NodeID]; !ok {
				nodeMap[summary.NodeID] = nil
			}
		}
		tasks = append(tasks, task)
	}

	// Get nodes
	nodes, err := dep.NodeClient().ListActiveNodes(c, lo.Keys(nodeMap))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to query nodes", err)
	}
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	// Build response
	return BuildTaskListResponse(tasks, res, nodeMap, hasher), nil
}

func TaskPhaseProgress(c *gin.Context, taskID int) (queue.Progresses, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	r := dep.TaskRegistry()
	t, found := r.Get(taskID)
	if !found || (t.Owner().ID != u.ID && !inventory.EffectiveGroup(u).Permissions.Enabled(int(types.GroupPermissionIsAdmin))) {
		return queue.Progresses{}, nil
	}

	return t.Progress(c), nil
}

func CancelDownloadTask(c *gin.Context, taskID int) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	r := dep.TaskRegistry()
	t, found := r.Get(taskID)
	if !found || t.Owner().ID != u.ID {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", nil)
	}

	if downloadTask, ok := t.(*workflows.RemoteDownloadTask); ok {
		if err := downloadTask.CancelDownload(c); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to cancel download task", err)
		}
	}

	return nil
}

// queueForTaskType maps a persisted task type to the dependency queue that
// owns it. Slave-side task types are not retryable from the master UI.
func queueForTaskType(c *gin.Context, dep dependency.Dep, taskType string) (queue.Queue, error) {
	switch taskType {
	case queue.CreateArchiveTaskType, queue.ExtractArchiveTaskType, queue.RelocateTaskType, queue.ImportTaskType:
		return dep.IoIntenseQueue(c), nil
	case queue.RemoteDownloadTaskType:
		return dep.RemoteDownloadQueue(c), nil
	case queue.MediaMetaTaskType, queue.FullTextIndexTaskType, queue.FullTextDeleteTaskType,
		queue.FullTextRebuildTaskType, queue.FullTextCopyTaskType, queue.FullTextChangeOwnerTaskType:
		return dep.MediaMetaQueue(c), nil
	case queue.EntityRecycleRoutineTaskType, queue.ExplicitEntityRecycleTaskType, queue.UploadSentinelCheckTaskType:
		return dep.EntityRecycleQueue(c), nil
	}
	return nil, fmt.Errorf("task type %q is not retryable", taskType)
}

// RetryTask re-queues a failed task with its original args (#2823).
func RetryTask(c *gin.Context, taskID int) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	taskClient := dep.TaskClient()

	ctx := context.WithValue(c, inventory.LoadTaskUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
	model, err := taskClient.GetTaskByID(ctx, taskID)
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", err)
	}

	if model.UserTasks != u.ID && !inventory.EffectiveGroup(u).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", nil)
	}

	if model.Status != task.StatusError {
		return serializer.NewError(serializer.CodeParamErr, "Only failed tasks can be retried", nil)
	}

	resumed, err := queue.NewTaskFromModel(model)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to rebuild task", err)
	}

	q, err := queueForTaskType(c, dep, model.Type)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "Task type is not retryable", err)
	}

	if err := q.QueueTask(c, resumed); err != nil {
		return serializer.NewError(serializer.CodeCreateTaskError, "Failed to queue task", err)
	}

	return nil
}

// CancelTask terminates a queued or suspending task; running remote
// downloads are canceled through their downloader handle (#2270).
func CancelTask(c *gin.Context, taskID int) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	taskClient := dep.TaskClient()

	ctx := context.WithValue(c, inventory.LoadTaskUser{}, true)
	model, err := taskClient.GetTaskByID(ctx, taskID)
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", err)
	}

	if model.UserTasks != u.ID && !inventory.EffectiveGroup(u).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", nil)
	}

	switch model.Status {
	case task.StatusQueued, task.StatusSuspending:
		q, err := queueForTaskType(c, dep, model.Type)
		if err != nil {
			return serializer.NewError(serializer.CodeParamErr, "Task type cannot be canceled", err)
		}
		if q.CancelTask(ctx, taskID) {
			return nil
		}
		// Not in the in-memory registry (e.g. after a restart before
		// resume); persist the cancel so it won't be picked up later.
		if err := taskClient.SetStatusByID(ctx, taskID, task.StatusCanceled); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to cancel task", err)
		}
		return nil
	case task.StatusProcessing:
		// Only remote downloads expose a runtime cancel handle; other
		// running tasks cannot be interrupted safely.
		if t, found := dep.TaskRegistry().Get(taskID); found {
			if dl, ok := t.(*workflows.RemoteDownloadTask); ok {
				if err := dl.CancelDownload(c); err != nil {
					return serializer.NewError(serializer.CodeInternalSetting, "Failed to cancel download task", err)
				}
				return nil
			}
		}
		return serializer.NewError(serializer.CodeParamErr, "Running task cannot be canceled", nil)
	default:
		return serializer.NewError(serializer.CodeParamErr, "Only queued or running tasks can be canceled", nil)
	}
}

// DeleteTask hides a finished task record from its owner's list. The row
// is kept so admins retain the history (#2270 follow-up, user request).
func DeleteTask(c *gin.Context, taskID int) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	taskClient := dep.TaskClient()

	model, err := taskClient.GetTaskByID(c, taskID)
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", err)
	}

	if model.UserTasks != u.ID && !inventory.EffectiveGroup(u).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", nil)
	}

	switch model.Status {
	case task.StatusCompleted, task.StatusError, task.StatusCanceled:
	default:
		return serializer.NewError(serializer.CodeParamErr, "Only finished tasks can be deleted", nil)
	}

	if err := taskClient.HideByIDs(c, model.UserTasks, taskID); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete task", err)
	}
	return nil
}

type (
	SetDownloadFilesService struct {
		Files []*downloader.SetFileToDownloadArgs `json:"files" binding:"required"`
	}
	SetDownloadFilesParamCtx struct{}
)

func (service *SetDownloadFilesService) SetDownloadFiles(c *gin.Context, taskID int) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	r := dep.TaskRegistry()

	t, found := r.Get(taskID)
	if !found || t.Owner().ID != u.ID {
		return serializer.NewError(serializer.CodeNotFound, "Task not found", nil)
	}

	status := t.Status()
	summary := t.Summarize(dep.HashIDEncoder())
	// Task must be in processing state
	if status != task.StatusSuspending && status != task.StatusProcessing {
		return serializer.NewError(serializer.CodeNotFound, "Task not in processing state", nil)
	}

	// Task must in monitoring loop
	if summary.Phase != workflows.RemoteDownloadTaskPhaseMonitor {
		return serializer.NewError(serializer.CodeNotFound, "Task not in monitoring loop", nil)
	}

	if downloadTask, ok := t.(*workflows.RemoteDownloadTask); ok {
		if err := downloadTask.SetDownloadTarget(c, service.Files...); err != nil {
			return serializer.NewError(serializer.CodeInternalSetting, "Failed to set download files", err)
		}
	}

	return nil
}

type (
	RebuildFTSIndexWorkflowService struct {
		FilteredStoragePolicy []int `json:"filtered_storage_policy"`
	}
	CreateRebuildFTSIndexParamCtx struct{}
)

func (service *RebuildFTSIndexWorkflowService) CreateRebuildFTSIndexTask(c *gin.Context) (*TaskResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	if !inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Only admin can import files", nil)
	}

	// Create task
	t, err := workflows.NewRebuildIndexTask(c, user, service.FilteredStoragePolicy)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to create task", err)
	}

	if err := dep.MediaMetaQueue(c).QueueTask(c, t); err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to queue task", err)
	}

	return BuildTaskResponse(t, nil, hasher), nil
}

type (
	BlobAuditWorkflowService struct {
		PolicyID int  `json:"policy_id" binding:"required,min=1"`
		Delete   bool `json:"delete"`
	}
	BlobAuditParamCtx struct{}
)

// CreateBlobAuditTask queues a blob-vs-database audit for one storage policy.
func (service *BlobAuditWorkflowService) CreateBlobAuditTask(c *gin.Context) (*TaskResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()

	if !inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, serializer.NewError(serializer.CodeGroupNotAllowed, "Only admin can run a blob audit", nil)
	}

	if _, err := dep.StoragePolicyClient().GetPolicyByID(c, service.PolicyID); err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Storage policy not found", err)
	}

	t, err := workflows.NewBlobAuditTask(c, user, service.PolicyID, service.Delete)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to create task", err)
	}

	if err := dep.IoIntenseQueue(c).QueueTask(c, t); err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to queue task", err)
	}

	return BuildTaskResponse(t, nil, hasher), nil
}

// providerNodeAvailable returns the ID of an active remote-download node
// offering the given provider, or 0 when none does.
func providerNodeAvailable(c *gin.Context, dep dependency.Dep, provider string) int {
	nodes, err := dep.NodeClient().ListActiveNodes(c, nil)
	if err != nil {
		return 0
	}
	return workflows.PickProviderNode(nodes, provider)
}
