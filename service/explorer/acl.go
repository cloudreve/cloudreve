package explorer

import (
	"fmt"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/aclentry"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// AclEntryResponse is one row in the Permissions dialog.
type AclEntryResponse struct {
	ID          int      `json:"id"`
	SubjectType string   `json:"subject_type"`
	SubjectID   int      `json:"subject_id"`
	SubjectName string   `json:"subject_name"`
	Permissions []string `json:"permissions"`
}

// AclSubjectResponse is a selectable subject in the Permissions dialog.
type AclSubjectResponse struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type (
	// AclListService lists ACL entries of a file.
	AclListService struct {
		Uri string `form:"uri" binding:"required"`
	}
	AclListParamCtx struct{}

	// AclUpsertService creates or updates an ACL entry.
	AclUpsertService struct {
		Uri         string   `json:"uri" binding:"required"`
		SubjectType string   `json:"subject_type" binding:"required,oneof=user group anonymous everyone"`
		SubjectID   int      `json:"subject_id"`
		Permissions []string `json:"permissions"`
	}
	AclUpsertParamCtx struct{}

	// AclDeleteService removes an ACL entry.
	AclDeleteService struct {
		Uri string `form:"uri" binding:"required"`
		ID  int    `form:"id" binding:"required,min=1"`
	}
	AclDeleteParamCtx struct{}

	// AclSubjectSearchService resolves selectable subjects for the dialog.
	AclSubjectSearchService struct {
		Keyword string `form:"keyword"`
	}
	AclSubjectSearchParamCtx struct{}
)

func aclPermissionsToSet(perms []string) (*boolset.BooleanSet, error) {
	bs := &boolset.BooleanSet{}
	nameToBit := lo.Invert(types.AclPermissionList)
	for _, p := range perms {
		bit, ok := nameToBit[p]
		if !ok {
			return nil, fmt.Errorf("unknown permission %q", p)
		}
		boolset.Set(int(bit), true, bs)
	}
	return bs, nil
}

func aclSetToPermissions(bs *boolset.BooleanSet) []string {
	res := []string{}
	for bit, name := range types.AclPermissionList {
		if bs != nil && bs.Enabled(int(bit)) {
			res = append(res, name)
		}
	}
	return res
}

// ownedAclFile resolves the target file and enforces that the acting user
// owns it and their group may manage explicit permissions.
func ownedAclFile(c *gin.Context, uriRaw string) (int, error) {
	user := inventory.UserFromContext(c)
	uri, err := fs.NewUriFromString(uriRaw)
	if err != nil {
		return 0, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	m := manager.NewFileManager(dependency.FromContext(c), user)
	defer m.Recycle()

	file, err := m.Get(c, uri)
	if err != nil {
		return 0, fmt.Errorf("failed to get file: %w", err)
	}
	if file.OwnerID() != user.ID {
		return 0, serializer.NewError(serializer.CodeNoPermissionErr, "Only the owner can manage permissions", nil)
	}

	if inventory.EffectiveGroup(user) == nil || inventory.EffectiveGroup(user).Permissions == nil ||
		!inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionSetExplicitUser)) {
		return 0, serializer.NewError(serializer.CodeNoPermissionErr, "Permission management is not enabled for your group", nil)
	}

	return file.ID(), nil
}

func (s *AclListService) Get(c *gin.Context) ([]*AclEntryResponse, error) {
	dep := dependency.FromContext(c)
	fileID, err := ownedAclFile(c, s.Uri)
	if err != nil {
		return nil, err
	}

	entries, err := dep.AclClient().List(c, fileID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list permissions", err)
	}

	// Resolve subject display names.
	names := map[string]string{}
	for _, e := range entries {
		key := fmt.Sprintf("%s:%d", e.SubjectType, e.SubjectID)
		switch e.SubjectType {
		case aclentry.SubjectTypeUser:
			if u, err := dep.UserClient().GetByID(c, e.SubjectID); err == nil {
				names[key] = u.Email
			}
		case aclentry.SubjectTypeGroup:
			if g, err := dep.GroupClient().GetByID(c, e.SubjectID); err == nil {
				names[key] = g.Name
			}
		}
	}

	return lo.Map(entries, func(e *ent.AclEntry, _ int) *AclEntryResponse {
		name := names[fmt.Sprintf("%s:%d", e.SubjectType, e.SubjectID)]
		if e.SubjectType == aclentry.SubjectTypeAnonymous {
			name = "anonymous"
		} else if e.SubjectType == aclentry.SubjectTypeEveryone {
			name = "everyone"
		}
		return &AclEntryResponse{
			ID:          e.ID,
			SubjectType: string(e.SubjectType),
			SubjectID:   e.SubjectID,
			SubjectName: name,
			Permissions: aclSetToPermissions(e.Permissions),
		}
	}), nil
}

func (s *AclUpsertService) Update(c *gin.Context) (*AclEntryResponse, error) {
	dep := dependency.FromContext(c)
	fileID, err := ownedAclFile(c, s.Uri)
	if err != nil {
		return nil, err
	}

	perms, err := aclPermissionsToSet(s.Permissions)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid permissions", err)
	}

	subjectType := aclentry.SubjectType(s.SubjectType)
	if (subjectType == aclentry.SubjectTypeAnonymous || subjectType == aclentry.SubjectTypeEveryone) && s.SubjectID != 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "General access entries must not carry a subject id", nil)
	}
	if (subjectType == aclentry.SubjectTypeUser || subjectType == aclentry.SubjectTypeGroup) && s.SubjectID <= 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Subject id is required", nil)
	}

	e, err := dep.AclClient().Upsert(c, &inventory.UpsertAclEntryParams{
		FileID:      fileID,
		SubjectType: subjectType,
		SubjectID:   s.SubjectID,
		Permissions: perms,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to save permission", err)
	}

	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventSetFilePermission,
		activity.File(fileID),
		activity.Extra(map[string]any{"subject_type": string(subjectType), "subject_id": s.SubjectID, "permissions": s.Permissions}))

	return &AclEntryResponse{
		ID:          e.ID,
		SubjectType: string(e.SubjectType),
		SubjectID:   e.SubjectID,
		Permissions: aclSetToPermissions(e.Permissions),
	}, nil
}

func (s *AclDeleteService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	fileID, err := ownedAclFile(c, s.Uri)
	if err != nil {
		return err
	}

	if err := dep.AclClient().Delete(c, fileID, s.ID); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete permission", err)
	}
	activity.Record(c, dep.SettingProvider(), dep.ActivityClient(), types.EventSetFilePermission,
		activity.File(fileID), activity.Extra(map[string]any{"deleted_entry": s.ID}))
	return nil
}

func (s *AclSubjectSearchService) Get(c *gin.Context) ([]*AclSubjectResponse, error) {
	user := inventory.UserFromContext(c)
	if inventory.EffectiveGroup(user) == nil || inventory.EffectiveGroup(user).Permissions == nil ||
		!inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionSetExplicitUser)) {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Permission management is not enabled for your group", nil)
	}

	dep := dependency.FromContext(c)
	res := []*AclSubjectResponse{}

	// Exact email match only — user enumeration by prefix is not exposed.
	if s.Keyword != "" {
		if u, err := dep.UserClient().GetByEmail(c, s.Keyword); err == nil {
			res = append(res, &AclSubjectResponse{Type: "user", ID: u.ID, Name: u.Email})
		}
	}

	groups, err := dep.GroupClient().ListAll(c)
	if err == nil {
		for _, g := range groups {
			if s.Keyword == "" || strings.Contains(strings.ToLower(g.Name), strings.ToLower(s.Keyword)) {
				res = append(res, &AclSubjectResponse{Type: "group", ID: g.ID, Name: g.Name})
			}
		}
	}

	return res, nil
}
