package explorer

import (
	"errors"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/wopi"
	"github.com/gin-gonic/gin"
)

type WopiService struct {
}

func prepareFs(c *gin.Context) (*fs.URI, manager.FileManager, *ent.User, *manager.ViewerSessionCache, dependency.Dep, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	viewerSession := manager.ViewerSessionFromContext(c)
	uri, err := fs.NewUriFromString(viewerSession.Uri)
	if err != nil {
		return nil, nil, nil, nil, nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	return uri, m, user, viewerSession, dep, nil
}

func (service *WopiService) Unlock(c *gin.Context) error {
	_, m, _, _, dep, err := prepareFs(c)
	if err != nil {
		return err
	}

	l := dep.Logger()

	lockToken := c.GetHeader(wopi.LockTokenHeader)
	if err = m.Unlock(c, lockToken); err != nil {
		l.Debug("WOPI unlock, not locked or not match: %w", err)
		c.Status(http.StatusConflict)
		c.Header(wopi.LockTokenHeader, "")
		return nil
	}

	return nil
}

func (service *WopiService) RefreshLock(c *gin.Context) error {
	uri, m, _, _, dep, err := prepareFs(c)
	if err != nil {
		return err
	}

	l := dep.Logger()

	// Make sure file exists and readable
	file, err := m.Get(c, uri, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityLockFile), dbfs.WithNotRoot())
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	lockToken := c.GetHeader(wopi.LockTokenHeader)
	release, _, err := m.ConfirmLock(c, file, file.Uri(false), lockToken)
	if err != nil {
		// File not locked for token not match

		l.Debug("WOPI refresh lock, not locked or not match: %w", err)
		c.Status(http.StatusConflict)
		c.Header(wopi.LockTokenHeader, "")
		return nil

	}

	// refresh lock
	release()
	_, err = m.Refresh(c, wopi.LockDuration, lockToken)
	if err != nil {
		return err
	}

	c.Header(wopi.LockTokenHeader, lockToken)
	return nil
}

func (service *WopiService) Lock(c *gin.Context) error {
	uri, m, user, viewerSession, dep, err := prepareFs(c)
	if err != nil {
		return err
	}

	l := dep.Logger()

	// Make sure file exists and readable
	file, err := m.Get(c, uri, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityLockFile), dbfs.WithNotRoot())
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	lockToken := c.GetHeader(wopi.LockTokenHeader)
	release, _, err := m.ConfirmLock(c, file, file.Uri(false), lockToken)
	if err != nil {
		// File not locked for token not match

		// Try to lock using given token
		app := lock.Application{
			Type:     string(fs.ApplicationViewer),
			ViewerID: viewerSession.ViewerID,
		}
		_, err = m.Lock(c, wopi.LockDuration, user, true, app, file.Uri(false), lockToken)
		if err != nil {
			// Token not match
			var lockConflict lock.ConflictError
			if errors.As(err, &lockConflict) {
				c.Status(http.StatusConflict)
				c.Header(wopi.LockTokenHeader, lockConflict[0].Token)

				l.Debug("WOPI lock, lock conflict: %w", err)
				return nil
			}

			return fmt.Errorf("failed to lock file: %w", err)
		}

		// Lock success, return the token
		c.Header(wopi.LockTokenHeader, lockToken)
		return nil

	}

	// refresh lock
	release()
	_, err = m.Refresh(c, wopi.LockDuration, lockToken)
	if err != nil {
		return err
	}

	c.Header(wopi.LockTokenHeader, lockToken)
	return nil
}

func (service *WopiService) PutContent(c *gin.Context, isPutRelative bool) error {
	uri, m, user, viewerSession, _, err := prepareFs(c)
	if err != nil {
		return err
	}

	// Make sure file exists and readable
	file, err := m.Get(c, uri, dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityUploadFile), dbfs.WithNotRoot())
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	var lockSession fs.LockSession
	lockToken := c.GetHeader(wopi.LockTokenHeader)
	if lockToken != "" {
		// File not locked for token not match

		release, ls, err := m.ConfirmLock(c, file, file.Uri(false), lockToken)
		if err != nil {
			// File not locked for token not match

			// Try to lock using given token
			app := lock.Application{
				Type:     string(fs.ApplicationViewer),
				ViewerID: viewerSession.ViewerID,
			}
			ls, err := m.Lock(c, wopi.LockDuration, user, true, app, file.Uri(false), lockToken)
			if err != nil {
				// Token not match
				// If the file is currently locked and the X-WOPI-Lock value doesn't match the lock currently on the file, the host must
				//
				// Return a lock mismatch response (409 Conflict)
				// Include an X-WOPI-Lock response header containing the value of the current lock on the file.
				var lockConflict lock.ConflictError
				if errors.As(err, &lockConflict) {
					c.Status(http.StatusConflict)
					c.Header(wopi.LockTokenHeader, lockConflict[0].Token)

					return nil
				}

				return fmt.Errorf("failed to lock file: %w", err)
			}

			// In cases where the file is unlocked, the host must set X-WOPI-Lock to the empty string.
			c.Header(wopi.LockTokenHeader, "")
			_ = m.Unlock(c, ls.LastToken())
		} else {
			defer release()
		}

		lockSession = ls
	}

	fileUri := viewerSession.Uri
	if isPutRelative {
		// If the header contains only a file extension (starts with a period), then the resulting file name will consist of this extension and the initial file name without extension.
		// If the header contains a full file name, then it will be a name for the resulting file.
		fileName, err := wopi.UTF7Decode(c.GetHeader(wopi.SuggestedTargetHeader))
		if err != nil {
			return fmt.Errorf("failed to decode X-WOPI-SuggestedTarget header (UTF-7): %w", err)
		}

		fileUriParsed, err := fs.NewUriFromString(fileUri)
		if err != nil {
			return fmt.Errorf("failed to parse file uri: %w", err)
		}

		if strings.HasPrefix(fileName, ".") {
			fileName = strings.TrimSuffix(fileUriParsed.Name(), filepath.Ext(fileUriParsed.Name())) + fileName
		}
		fileUri = fileUriParsed.DirUri().JoinRaw(fileName).String()
	}

	subService := FileUpdateService{
		Uri: fileUri,
	}

	res, err := subService.PutContent(c, lockSession)
	if err != nil {
		var appErr serializer.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code {
			case serializer.CodeFileTooLarge:
				c.Status(http.StatusRequestEntityTooLarge)
				c.Header(wopi.ServerErrorHeader, err.Error())
			case serializer.CodeNotFound:
				c.Status(http.StatusNotFound)
				c.Header(wopi.ServerErrorHeader, err.Error())
			case 0:
				c.Status(http.StatusOK)
			default:
				return err
			}

			return nil
		}

		return err
	}

	c.Header(wopi.ItemVersionHeader, res.PrimaryEntity)
	if isPutRelative {
		c.JSON(http.StatusOK, PutRelativeResponse{
			Name: res.Name,
		})
	}
	return nil
}

func (service *WopiService) GetFile(c *gin.Context) error {
	uri, m, _, viewerSession, dep, err := prepareFs(c)
	if err != nil {
		return err
	}

	// Make sure file exists and readable
	file, err := m.Get(c, uri, dbfs.WithExtendedInfo(), dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile), dbfs.WithNotRoot())
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	versionType := types.EntityTypeVersion
	find, targetEntity := fs.FindDesiredEntity(file, viewerSession.Version, dep.HashIDEncoder(), &versionType)
	if !find {
		return serializer.NewError(serializer.CodeNotFound, "version not found", nil)
	}

	if targetEntity.Size() > dep.SettingProvider().MaxOnlineEditSize(c) {
		return fs.ErrFileSizeTooBig
	}

	entitySource, err := m.GetEntitySource(c, targetEntity.ID(), fs.WithEntity(targetEntity))
	if err != nil {
		return fmt.Errorf("failed to get entity source: %w", err)
	}

	defer entitySource.Close()

	entitySource.Serve(c.Writer, c.Request,
		entitysource.WithContext(c),
	)

	return nil
}

func (service *WopiService) FileInfo(c *gin.Context) (*WopiFileInfo, error) {
	uri, m, user, viewerSession, dep, err := prepareFs(c)
	if err != nil {
		return nil, err
	}

	hasher := dep.HashIDEncoder()
	settings := dep.SettingProvider()

	opts := []fs.Option{
		dbfs.WithFilePublicMetadata(),
		dbfs.WithExtendedInfo(),
		dbfs.WithNotRoot(),
		dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile, dbfs.NavigatorCapabilityInfo),
	}
	file, err := m.Get(c, uri, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if file == nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "file not found", nil)
	}

	versionType := types.EntityTypeVersion
	find, targetEntity := fs.FindDesiredEntity(file, viewerSession.Version, hasher, &versionType)
	if !find {
		return nil, serializer.NewError(serializer.CodeNotFound, "version not found", nil)
	}

	canEdit := file.PrimaryEntityID() == targetEntity.ID() && file.OwnerID() == user.ID && uri.FileSystem() == constants.FileSystemMy
	cantPutRelative := !canEdit
	siteUrl := settings.SiteURL(c)
	info := &WopiFileInfo{
		BaseFileName:            file.DisplayName(),
		Version:                 hashid.EncodeEntityID(hasher, targetEntity.ID()),
		BreadcrumbBrandName:     settings.SiteBasic(c).Name,
		BreadcrumbBrandUrl:      siteUrl.String(),
		FileSharingPostMessage:  file.OwnerID() == user.ID,
		EnableShare:             file.OwnerID() == user.ID,
		FileVersionPostMessage:  true,
		ClosePostMessage:        true,
		PostMessageOrigin:       "*",
		FileNameMaxLength:       dbfs.MaxFileNameLength,
		LastModifiedTime:        file.UpdatedAt().Format(time.RFC3339),
		IsAnonymousUser:         inventory.IsAnonymousUser(user),
		UserFriendlyName:        user.Nick,
		UserId:                  hashid.EncodeUserID(hasher, user.ID),
		ReadOnly:                !canEdit,
		Size:                    targetEntity.Size(),
		OwnerId:                 hashid.EncodeUserID(hasher, file.OwnerID()),
		SupportsRename:          true,
		SupportsReviewing:       true,
		SupportsLocks:           true,
		UserCanReview:           canEdit,
		UserCanWrite:            canEdit,
		UserCanNotWriteRelative: cantPutRelative,
		BreadcrumbFolderName:    uri.Dir(),
		BreadcrumbFolderUrl:     routes.FrontendHomeUrl(siteUrl, uri.DirUri().String()).String(),
	}

	return info, nil
}

type (
	CreateViewerSessionService struct {
		Uri             string               `json:"uri" form:"uri" binding:"required"`
		Version         string               `json:"version" form:"version"`
		ViewerID        string               `json:"viewer_id" form:"viewer_id" binding:"required"`
		PreferredAction types.ViewerAction `json:"preferred_action" form:"preferred_action" binding:"required"`
	}
	CreateViewerSessionParamCtx struct{}
)

func (s *CreateViewerSessionService) Create(c *gin.Context) (*ViewerSessionResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	// Find the given viewer
	viewers := dep.SettingProvider().FileViewers(c)
	var targetViewer *types.Viewer
	for _, group := range viewers {
		for _, viewer := range group.Viewers {
			if viewer.ID == s.ViewerID && !viewer.Disabled {
				targetViewer = &viewer
				break
			}
		}

		if targetViewer != nil {
			break
		}
	}

	if targetViewer == nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown viewer id", err)
	}

	viewerSession, err := m.CreateViewerSession(c, uri, s.Version, targetViewer)
	if err != nil {
		return nil, err
	}

	res := &ViewerSessionResponse{Session: viewerSession}
	if targetViewer.Type == types.ViewerTypeWopi {
		// For WOPI viewer, generate WOPI src
		wopiSrc, err := wopi.GenerateWopiSrc(c, s.PreferredAction, targetViewer, viewerSession)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "failed to generate wopi src", err)
		}
		res.WopiSrc = wopiSrc.String()
	}

	return res, nil
}
