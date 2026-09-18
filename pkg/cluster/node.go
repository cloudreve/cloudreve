package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/node"
	"github.com/cloudreve/Cloudreve/v4/ent/task"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader/aria2"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader/qbittorrent"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader/slave"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"strconv"
)

type (
	Node interface {
		fs.StatelessUploadManager
		ID() int
		Name() string
		IsMaster() bool
		// CreateTask creates a task on the node. It does not have effect on master node.
		CreateTask(ctx context.Context, taskType string, state string) (int, error)
		// GetTask returns the task summary of the task with the given id.
		GetTask(ctx context.Context, id int, clearOnComplete bool) (*SlaveTaskSummary, error)
		// CleanupFolders cleans up the given folders on the node.
		CleanupFolders(ctx context.Context, folders ...string) error
		// AuthInstance returns the auth instance for the node.
		AuthInstance() auth.Auth
		// CreateDownloader creates a downloader instance from the node for remote download tasks.
		CreateDownloader(ctx context.Context, c request.Client, settings setting.Provider) (downloader.Downloader, error)
		// Settings returns the settings of the node.
		Settings(ctx context.Context) *types.NodeSetting
	}

	// Request body for creating tasks on slave node
	CreateSlaveTask struct {
		Type  string `json:"type"`
		State string `json:"state"`
	}

	// Request body for cleaning up folders on slave node
	FolderCleanup struct {
		Path []string `json:"path" binding:"required"`
	}

	SlaveTaskSummary struct {
		Status       task.Status      `json:"status"`
		Error        string           `json:"error"`
		PrivateState string           `json:"private_state"`
		Progress     queue.Progresses `json:"progress,omitempty"`
	}

	MasterSiteUrlCtx     struct{}
	MasterSiteVersionCtx struct{}
	MasterSiteIDCtx      struct{}
	SlaveNodeIDCtx       struct{}
	masterNode           struct {
		nodeBase
		client request.Client
	}
)

func newNode(ctx context.Context, model *ent.Node, config conf.ConfigProvider, settings setting.Provider) Node {
	if model.Type == node.TypeMaster {
		return newMasterNode(model, config, settings)
	}
	return newSlaveNode(ctx, model, config, settings)
}

func newMasterNode(model *ent.Node, config conf.ConfigProvider, settings setting.Provider) *masterNode {
	n := &masterNode{
		nodeBase: nodeBase{
			model: model,
		},
	}

	if config.System().Mode == conf.SlaveMode {
		n.client = request.NewClient(config,
			request.WithCorrelationID(),
			request.WithCredential(auth.HMACAuth{SecretKey: []byte(config.Slave().Secret)}, int64(config.Slave().SignatureTTL)),
		)
	}

	return n
}

func (b *masterNode) PrepareUpload(ctx context.Context, args *fs.StatelessPrepareUploadService) (*fs.StatelessPrepareUploadResponse, error) {
	reqBody, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	requestDst := routes.MasterStatelessUrl(MasterSiteUrlFromContext(ctx), "prepare")
	resp, err := b.client.Request(
		"PUT",
		requestDst.String(),
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithSlaveMeta(NodeIdFromContext(ctx)),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return nil, serializer.NewErrorFromResponse(resp)
	}

	uploadRequest := &fs.StatelessPrepareUploadResponse{}
	resp.DecodeGob(uploadRequest)

	return uploadRequest, nil
}

func (b *masterNode) CompleteUpload(ctx context.Context, args *fs.StatelessCompleteUploadService) error {
	reqBody, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	requestDst := routes.MasterStatelessUrl(MasterSiteUrlFromContext(ctx), "complete")
	resp, err := b.client.Request(
		"POST",
		requestDst.String(),
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithSlaveMeta(NodeIdFromContext(ctx)),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (b *masterNode) OnUploadFailed(ctx context.Context, args *fs.StatelessOnUploadFailedService) error {
	reqBody, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	requestDst := routes.MasterStatelessUrl(MasterSiteUrlFromContext(ctx), "failed")
	resp, err := b.client.Request(
		"POST",
		requestDst.String(),
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithSlaveMeta(NodeIdFromContext(ctx)),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (b *masterNode) CreateFile(ctx context.Context, args *fs.StatelessCreateFileService) error {
	reqBody, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	requestDst := routes.MasterStatelessUrl(MasterSiteUrlFromContext(ctx), "create")
	resp, err := b.client.Request(
		"POST",
		requestDst.String(),
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithSlaveMeta(NodeIdFromContext(ctx)),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil

}

func (b *masterNode) CreateDownloader(ctx context.Context, c request.Client, settings setting.Provider) (downloader.Downloader, error) {
	return NewDownloader(ctx, c, settings, b.Settings(ctx))
}

// NewDownloader creates a new downloader instance from the node for remote download tasks.
func NewDownloader(ctx context.Context, c request.Client, settings setting.Provider, options *types.NodeSetting) (downloader.Downloader, error) {
	if options.Provider == types.DownloaderProviderQBittorrent {
		return qbittorrent.NewClient(logging.FromContext(ctx), c, settings, options.QBittorrentSetting)
	} else if options.Provider == types.DownloaderProviderAria2 {
		return aria2.New(logging.FromContext(ctx), settings, options.Aria2Setting), nil
	} else if options.Provider == "" {
		return nil, errors.New("downloader not configured for this node")
	} else {
		return nil, errors.New("unknown downloader provider")
	}
}

type slaveNode struct {
	nodeBase
	client request.Client
}

func newSlaveNode(ctx context.Context, model *ent.Node, config conf.ConfigProvider, settings setting.Provider) *slaveNode {
	siteBasic := settings.SiteBasic(ctx)
	return &slaveNode{
		nodeBase: nodeBase{
			model: model,
		},
		client: request.NewClient(config,
			request.WithCorrelationID(),
			request.WithSlaveMeta(model.ID),
			request.WithMasterMeta(siteBasic.ID, settings.SiteURL(setting.UseFirstSiteUrl(ctx)).String()),
			request.WithCredential(auth.HMACAuth{SecretKey: []byte(model.SlaveKey)}, int64(settings.SlaveRequestSignTTL(ctx))),
			request.WithEndpoint(model.Server)),
	}
}

func (n *slaveNode) CreateTask(ctx context.Context, taskType string, state string) (int, error) {
	reqBody, err := json.Marshal(&CreateSlaveTask{
		Type:  taskType,
		State: state,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := n.client.Request(
		"PUT",
		constants.APIPrefixSlave+"/task",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return 0, err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return 0, serializer.NewErrorFromResponse(resp)
	}

	taskId := 0
	if resp.DecodeGob(&taskId); taskId > 0 {
		return taskId, nil
	}

	return 0, fmt.Errorf("unexpected response data: %v", resp.Data)
}

func (n *slaveNode) GetTask(ctx context.Context, id int, clearOnComplete bool) (*SlaveTaskSummary, error) {
	resp, err := n.client.Request(
		"GET",
		routes.SlaveGetTaskRoute(id, clearOnComplete),
		nil,
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return nil, serializer.NewErrorFromResponse(resp)
	}

	summary := &SlaveTaskSummary{}
	resp.DecodeGob(summary)

	return summary, nil
}

func (b *slaveNode) CleanupFolders(ctx context.Context, folders ...string) error {
	args := &FolderCleanup{
		Path: folders,
	}
	reqBody, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := b.client.Request(
		"POST",
		constants.APIPrefixSlave+"/task/cleanup",
		bytes.NewReader(reqBody),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	// 处理列取结果
	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (b *slaveNode) CreateDownloader(ctx context.Context, c request.Client, settings setting.Provider) (downloader.Downloader, error) {
	return slave.NewSlaveDownloader(b.client, b.Settings(ctx)), nil
}

type nodeBase struct {
	model *ent.Node
}

func (b *nodeBase) ID() int {
	return b.model.ID
}

func (b *nodeBase) Name() string {
	return b.model.Name
}

func (b *nodeBase) IsMaster() bool {
	return b.model.Type == node.TypeMaster
}

func (b *nodeBase) CreateTask(ctx context.Context, taskType string, state string) (int, error) {
	return 0, errors.New("not implemented")
}

func (b *nodeBase) AuthInstance() auth.Auth {
	return auth.HMACAuth{SecretKey: []byte(b.model.SlaveKey)}
}

func (b *nodeBase) GetTask(ctx context.Context, id int, clearOnComplete bool) (*SlaveTaskSummary, error) {
	return nil, errors.New("not implemented")
}

func (b *nodeBase) CleanupFolders(ctx context.Context, folders ...string) error {
	return errors.New("not implemented")
}

func (b *nodeBase) PrepareUpload(ctx context.Context, args *fs.StatelessPrepareUploadService) (*fs.StatelessPrepareUploadResponse, error) {
	return nil, errors.New("not implemented")
}

func (b *nodeBase) CompleteUpload(ctx context.Context, args *fs.StatelessCompleteUploadService) error {
	return errors.New("not implemented")
}

func (b *nodeBase) OnUploadFailed(ctx context.Context, args *fs.StatelessOnUploadFailedService) error {
	return errors.New("not implemented")
}

func (b *nodeBase) CreateFile(ctx context.Context, args *fs.StatelessCreateFileService) error {
	return errors.New("not implemented")
}

func (b *nodeBase) CreateDownloader(ctx context.Context, c request.Client, settings setting.Provider) (downloader.Downloader, error) {
	return nil, errors.New("not implemented")
}

func (b *nodeBase) Settings(ctx context.Context) *types.NodeSetting {
	return b.model.Settings
}

func NodeIdFromContext(ctx context.Context) int {
	nodeIdStr, ok := ctx.Value(SlaveNodeIDCtx{}).(string)
	if !ok {
		return 0
	}

	nodeId, _ := strconv.Atoi(nodeIdStr)
	return nodeId
}

func MasterSiteUrlFromContext(ctx context.Context) string {
	if u, ok := ctx.Value(MasterSiteUrlCtx{}).(string); ok {
		return u
	}

	return ""
}
