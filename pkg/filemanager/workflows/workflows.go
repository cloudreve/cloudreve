package workflows

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/queue"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

const (
	TaskTempPath                 = "fm_workflows"
	slaveProgressRefreshInterval = 5 * time.Second
)

type NodeState struct {
	NodeID int `json:"node_id"`
	// AllowedNodes restricts auto-dispatch to the group's node pool, captured
	// at task creation so later group edits don't reroute queued tasks.
	AllowedNodes []int `json:"allowed_nodes,omitempty"`

	progress queue.Progresses
}

// NodeSelection carries the caller-side node constraints for a new task:
// TargetNodeID is an explicit user pick (0 = auto), AllowedNodes is the
// group's eligible pool (empty = all).
type NodeSelection struct {
	TargetNodeID int
	AllowedNodes []int
}

func (s NodeSelection) state() NodeState {
	return NodeState{NodeID: s.TargetNodeID, AllowedNodes: s.AllowedNodes}
}

// allocateNode allocates a node for the task.
func allocateNode(ctx context.Context, dep dependency.Dep, state *NodeState, capability types.NodeCapability) (cluster.Node, error) {
	np, err := dep.NodePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node pool: %w", err)
	}

	node, err := np.Get(ctx, capability, state.NodeID, state.AllowedNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	state.NodeID = node.ID()
	return node, nil
}

// prepareSlaveTaskCtx prepares the context for the slave task.
func prepareSlaveTaskCtx(ctx context.Context, props *types.SlaveTaskProps) context.Context {
	ctx = context.WithValue(ctx, cluster.SlaveNodeIDCtx{}, strconv.Itoa(props.NodeID))
	ctx = context.WithValue(ctx, cluster.MasterSiteUrlCtx{}, props.MasterSiteURl)
	ctx = context.WithValue(ctx, cluster.MasterSiteVersionCtx{}, props.MasterSiteVersion)
	ctx = context.WithValue(ctx, cluster.MasterSiteIDCtx{}, props.MasterSiteID)
	return ctx
}

func prepareTempFolder(ctx context.Context, dep dependency.Dep, t queue.Task) (string, error) {
	settings := dep.SettingProvider()
	tempPath := util.DataPath(path.Join(settings.TempPath(ctx), TaskTempPath, strconv.Itoa(t.ID())))
	if err := util.CreatNestedFolder(tempPath); err != nil {
		return "", fmt.Errorf("failed to create temp folder: %w", err)
	}

	dep.Logger().Info("Temp folder created: %s", tempPath)
	return tempPath, nil
}

// preferPolicyNode seeds the preferred node with the node hosting the given
// file's storage policy, so data-heavy tasks run where the blob already
// lives. No-op when the URI is invalid, has no entity, or the policy is not
// bound to a node — the pool then falls back to weighted selection.
func preferPolicyNode(ctx context.Context, dep dependency.Dep, state *NodeState, uriStr string) {
	if state.NodeID > 0 {
		return
	}

	uri, err := fs.NewUriFromString(uriStr)
	if err != nil {
		return
	}

	fm := manager.NewFileManager(dep, inventory.UserFromContext(ctx))
	defer fm.Recycle()

	file, err := fm.Get(ctx, uri, dbfs.WithFileEntities())
	if err != nil || file == nil {
		return
	}

	entity := file.PrimaryEntity()
	if entity == nil {
		return
	}

	policy, err := dep.StoragePolicyClient().GetPolicyByID(ctx, entity.PolicyID())
	if err != nil || policy == nil {
		return
	}

	state.NodeID = policy.NodeID
}
