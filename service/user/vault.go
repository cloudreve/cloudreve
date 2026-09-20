package user

import (
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

const (
	// VaultFolderName is the default name of the private-space root folder.
	VaultFolderName = "Private space"
)

type (
	// VaultSetupParameterCtx marks the private-space setup route.
	VaultSetupParameterCtx struct{}
	// VaultUnlockParameterCtx marks the private-space unlock/lock routes.
	VaultUnlockParameterCtx struct{}
	// VaultDisableParameterCtx marks the private-space disable route.
	VaultDisableParameterCtx struct{}

	// VaultSetupService enables the private space for the current user by
	// setting an independent vault password and creating the vault folder.
	VaultSetupService struct {
		Password string `json:"password" binding:"required,min=6,max=128"`
	}

	// VaultUnlockService verifies the vault password and opens an unlock
	// session for subsequent file operations.
	VaultUnlockService struct {
		Password string `json:"password" binding:"required"`
	}

	// VaultDisableService turns off the private space. The vault folder and
	// its contents stay as regular files; only the gating is removed.
	VaultDisableService struct {
		Password string `json:"password" binding:"required"`
	}
)

// Setup enables the private space.
func (service *VaultSetupService) Setup(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u.VaultFolder > 0 {
		return serializer.NewError(serializer.CodeConflict, "private space is already enabled", nil)
	}

	digest, err := inventory.DigestPassword(service.Password)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to hash vault password", err)
	}

	m := manager.NewFileManager(dep, u)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(constants.CloudreveScheme + "://" + string(constants.FileSystemMy) + "/" + VaultFolderName)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to build vault path", err)
	}

	folder, err := m.Create(c, uri, types.FileTypeFolder)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to create private space folder", err)
	}

	folderModel, err := dep.FileClient().GetByID(c, folder.ID())
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to load private space folder", err)
	}

	if err := dep.FileClient().UpsertMetadata(c, folderModel, map[string]string{
		dbfs.MetadataVault: "1",
	}, nil); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to mark private space folder", err)
	}

	if _, err := dep.UserClient().UpdateVault(c, u, digest, folder.ID()); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to enable private space", err)
	}

	return nil
}

// Unlock verifies the vault password and opens the unlock session.
func (service *VaultUnlockService) Unlock(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u.VaultFolder <= 0 {
		return serializer.NewError(serializer.CodeNotFound, "private space is not enabled", nil)
	}

	if err := inventory.CheckVaultPassword(u, service.Password); err != nil {
		return serializer.NewError(serializer.CodeInvalidPassword, "Incorrect vault password", err)
	}

	if err := dep.KV().Set(dbfs.VaultUnlockCachePrefix+strconv.Itoa(u.ID), 1, dbfs.VaultUnlockTTL); err != nil {
		return serializer.NewError(serializer.CodeCacheOperation, "Failed to open vault session", err)
	}

	return nil
}

// Lock closes the unlock session immediately.
func (service *VaultUnlockService) Lock(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	dep.KV().Delete(dbfs.VaultUnlockCachePrefix, strconv.Itoa(u.ID))
	return nil
}

// Disable verifies the vault password and turns off the private space.
func (service *VaultDisableService) Disable(c *gin.Context) error {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	if u.VaultFolder <= 0 {
		return serializer.NewError(serializer.CodeNotFound, "private space is not enabled", nil)
	}

	if err := inventory.CheckVaultPassword(u, service.Password); err != nil {
		return serializer.NewError(serializer.CodeInvalidPassword, "Incorrect vault password", err)
	}

	if folder, err := dep.FileClient().GetByID(c, u.VaultFolder); err == nil && folder != nil {
		if err := dep.FileClient().RemoveMetadata(c, folder, dbfs.MetadataVault); err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to unmark private space folder", err)
		}
	}

	if _, err := dep.UserClient().UpdateVault(c, u, "", 0); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to disable private space", err)
	}

	dep.KV().Delete(dbfs.VaultUnlockCachePrefix, strconv.Itoa(u.ID))
	return nil
}
