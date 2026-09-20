package explorer

import (
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/workflows"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// StoragePolicyBrief is the user-facing view of an allowed storage policy.
type StoragePolicyBrief struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsDefault bool   `json:"is_default,omitempty"`
}

type (
	// AllowedPolicyService lists the storage policies the caller's group may use.
	AllowedPolicyService  struct{}
	AllowedPolicyParamCtx struct{}

	// PreferredPolicyService sets or clears a directory's preferred storage
	// policy. New uploads inside the directory use it.
	PreferredPolicyService struct {
		Uri    string `json:"uri" binding:"required"`
		Policy string `json:"policy"`
	}
	PreferredPolicyParamCtx struct{}

	// FileRelocateService moves a file's or folder's entities to another
	// storage policy via the resumable relocation task.
	FileRelocateService struct {
		Uri    string `json:"uri" binding:"required"`
		Policy string `json:"policy" binding:"required"`
	}
	FileRelocateParamCtx struct{}

	// FileRelocateResponse carries the created relocation task id.
	FileRelocateResponse struct {
		ID string `json:"id"`
	}
)

// allowedGroupPolicies resolves the union of usable policies across every
// group the user belongs to and maps each to a brief. The primary group's
// configured default is marked.
func allowedGroupPolicies(c *gin.Context, groups []*ent.Group, dep dependency.Dep) ([]*ent.StoragePolicy, []*StoragePolicyBrief, error) {
	allowed, err := dep.StoragePolicyClient().ListByGroups(c, groups)
	if err != nil {
		return nil, nil, serializer.NewError(serializer.CodeDBError, "Failed to list storage policies", err)
	}
	if len(allowed) == 0 {
		return nil, nil, serializer.NewError(serializer.CodeNoPermissionErr, "No storage policy is available for your group", nil)
	}

	defaultPolicyID := 0
	if len(groups) > 0 && groups[0] != nil {
		defaultPolicyID = groups[0].StoragePolicyID
	}
	hasher := dep.HashIDEncoder()
	briefs := lo.Map(allowed, func(p *ent.StoragePolicy, _ int) *StoragePolicyBrief {
		return &StoragePolicyBrief{
			ID:        hashid.EncodePolicyID(hasher, p.ID),
			Name:      p.Name,
			Type:      p.Type,
			IsDefault: p.ID == defaultPolicyID,
		}
	})
	return allowed, briefs, nil
}

func (s *AllowedPolicyService) Get(c *gin.Context) ([]*StoragePolicyBrief, error) {
	user := inventory.UserFromContext(c)
	if len(inventory.GroupsOf(user)) == 0 {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Group not loaded", nil)
	}
	_, briefs, err := allowedGroupPolicies(c, inventory.GroupsOf(user), dependency.FromContext(c))
	return briefs, err
}

// decodeAllowedPolicy validates a hashid-encoded policy id against the
// union of the user's group allowed sets and returns the matching policy.
func decodeAllowedPolicy(c *gin.Context, dep dependency.Dep, groups []*ent.Group, raw string) (*ent.StoragePolicy, error) {
	id, err := dep.HashIDEncoder().Decode(raw, hashid.PolicyID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid storage policy", err)
	}

	allowed, err := dep.StoragePolicyClient().ListByGroups(c, groups)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list storage policies", err)
	}
	for _, p := range allowed {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Storage policy is not available for your group", nil)
}

// ownedPolicyFile resolves the target and enforces ownership, returning the
// fs file for metadata access.
func ownedPolicyFile(c *gin.Context, dep dependency.Dep, uriRaw string) (*dbfs.File, error) {
	user := inventory.UserFromContext(c)
	uri, err := fs.NewUriFromString(uriRaw)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	f, err := m.Get(c, uri, dbfs.WithFileEntities())
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	file, ok := f.(*dbfs.File)
	if !ok {
		return nil, serializer.NewError(serializer.CodeParamErr, "Unsupported file system", nil)
	}
	if file.OwnerID() != user.ID {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Only the owner can manage storage policies", nil)
	}
	return file, nil
}

func (s *PreferredPolicyService) Update(c *gin.Context) (*StoragePolicyBrief, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	file, err := ownedPolicyFile(c, dep, s.Uri)
	if err != nil {
		return nil, err
	}
	if file.Type() != types.FileTypeFolder {
		return nil, serializer.NewError(serializer.CodeParamErr, "Preferred storage policy applies to folders only", nil)
	}

	var policy *ent.StoragePolicy
	if s.Policy != "" {
		policy, err = decodeAllowedPolicy(c, dep, inventory.GroupsOf(user), s.Policy)
		if err != nil {
			return nil, err
		}
	}

	fc := dep.FileClient()
	if policy == nil {
		err = fc.RemoveMetadata(c, file.Model, dbfs.MetadataPreferredPolicy)
	} else {
		err = fc.UpsertMetadata(c, file.Model, map[string]string{
			dbfs.MetadataPreferredPolicy: hashid.EncodePolicyID(dep.HashIDEncoder(), policy.ID),
		}, map[string]bool{dbfs.MetadataPreferredPolicy: true})
	}
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update metadata", err)
	}

	if policy == nil {
		return nil, nil
	}
	return &StoragePolicyBrief{
		ID:   hashid.EncodePolicyID(dep.HashIDEncoder(), policy.ID),
		Name: policy.Name,
		Type: policy.Type,
	}, nil
}

func (s *FileRelocateService) Create(c *gin.Context) (*FileRelocateResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)

	policy, err := decodeAllowedPolicy(c, dep, inventory.GroupsOf(user), s.Policy)
	if err != nil {
		return nil, err
	}
	// A load-balance target resolves to one concrete child at enqueue time.
	if policy.Type == types.PolicyTypeLoadBalance {
		policy, err = dep.StoragePolicyClient().ResolveLoadBalance(c, policy)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to resolve load-balanced storage policy", err)
		}
	}

	file, err := ownedPolicyFile(c, dep, s.Uri)
	if err != nil {
		return nil, err
	}

	// Depth is a countdown in the walk implementation; a large value walks
	// the whole subtree. Entities already on the destination are skipped by
	// the task, so they are filtered out of the request entirely.
	pending := lo.Filter(file.Model.Edges.Entities, func(e *ent.Entity, _ int) bool {
		return e.StoragePolicyEntities != policy.ID
	})
	entityIDs := lo.Map(pending, func(e *ent.Entity, _ int) int { return e.ID })
	if file.Type() == types.FileTypeFolder {
		uri, _ := fs.NewUriFromString(s.Uri)
		m := manager.NewFileManager(dep, user)
		defer m.Recycle()
		err = m.Walk(c, uri, 1<<30, func(f fs.File, _ int) error {
			dbFile, ok := f.(*dbfs.File)
			if !ok {
				return nil
			}
			for _, e := range dbFile.Model.Edges.Entities {
				if e.StoragePolicyEntities != policy.ID {
					entityIDs = append(entityIDs, e.ID)
				}
			}
			return nil
		}, dbfs.WithFileEntities())
		if err != nil {
			return nil, fmt.Errorf("failed to walk folder: %w", err)
		}
	}

	entityIDs = lo.Uniq(entityIDs)
	if len(entityIDs) == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Nothing to relocate", nil)
	}

	t, err := workflows.NewRelocateTask(c, entityIDs, policy.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeCreateTaskError, "Failed to create relocation task", err)
	}
	if err := dep.IoIntenseQueue(c).QueueTask(c, t); err != nil {
		return nil, fmt.Errorf("failed to submit task: %w", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventRelocate,
		activity.File(file.Model.ID),
		activity.Extra(map[string]any{"policy_id": policy.ID, "entities": len(entityIDs)}))
	return &FileRelocateResponse{ID: hashid.EncodeTaskID(dep.HashIDEncoder(), t.ID())}, nil
}
