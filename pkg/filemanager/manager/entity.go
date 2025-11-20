package manager

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
)

type (
	EntityManagement interface {
		// GetEntityUrls gets download urls of given entities, return URLs and the earliest expiry time
		GetEntityUrls(ctx context.Context, args []GetEntityUrlArgs, opts ...fs.Option) ([]EntityUrl, *time.Time, error)
		// GetUrlForRedirectedDirectLink gets redirected direct download link of given direct link
		GetUrlForRedirectedDirectLink(ctx context.Context, dl *ent.DirectLink, opts ...fs.Option) (string, *time.Time, error)
		// GetDirectLink gets permanent direct download link of given files
		GetDirectLink(ctx context.Context, urls ...*fs.URI) ([]DirectLink, error)
		// GetEntitySource gets source of given entity
		GetEntitySource(ctx context.Context, entityID int, opts ...fs.Option) (entitysource.EntitySource, error)
		// Thumbnail gets thumbnail entity of given file
		Thumbnail(ctx context.Context, uri *fs.URI) (entitysource.EntitySource, error)
		// SubmitAndAwaitThumbnailTask submits a thumbnail task and waits for result
		SubmitAndAwaitThumbnailTask(ctx context.Context, uri *fs.URI, ext string, entity fs.Entity) (fs.Entity, error)
		// SetCurrentVersion sets current version of given file
		SetCurrentVersion(ctx context.Context, path *fs.URI, version int) error
		// DeleteVersion deletes a version of given file
		DeleteVersion(ctx context.Context, path *fs.URI, version int) error
		// ExtractAndSaveMediaMeta extracts and saves media meta into file metadata of given file.
		ExtractAndSaveMediaMeta(ctx context.Context, uri *fs.URI, entityID int) error
		// RecycleEntities recycles a group of entities
		RecycleEntities(ctx context.Context, force bool, entityIDs ...int) error
		// ListPhysical lists physical files in a path
		ListPhysical(ctx context.Context, path string, policyID int, recursive bool, progress driver.ListProgressFunc) ([]fs.PhysicalObject, error)
		// ImportPhysical imports a physical file to a Cloudreve file
		ImportPhysical(ctx context.Context, dst *fs.URI, policyId int, src fs.PhysicalObject, completeHook bool) error
	}
	DirectLink struct {
		File fs.File
		Url  string
	}

	EntityUrl struct {
		Url                        string `json:"url"`
		BrowserDownloadDisplayName string `json:"stream_saver_display_name,omitempty"`
	}
)

func (m *manager) GetDirectLink(ctx context.Context, urls ...*fs.URI) ([]DirectLink, error) {
	ae := serializer.NewAggregateError()
	res := make([]DirectLink, 0, len(urls))
	useRedirect := m.user.Edges.Group.Settings.RedirectedSource
	fileClient := m.dep.FileClient()
	siteUrl := m.settings.SiteURL(ctx)

	for _, url := range urls {
		file, err := m.fs.Get(
			ctx, url,
			dbfs.WithFileEntities(),
			dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile),
		)
		if err != nil {
			ae.Add(url.String(), err)
			continue
		}

		if file.OwnerID() != m.user.ID && !m.user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
			ae.Add(url.String(), fs.ErrOwnerOnly)
			continue
		}

		if file.Type() != types.FileTypeFile {
			ae.Add(url.String(), fs.ErrEntityNotExist)
			continue
		}

		target := file.PrimaryEntity()
		if target == nil {
			ae.Add(url.String(), fs.ErrEntityNotExist)
			continue
		}

		// Hooks for entity download
		if err := m.fs.ExecuteNavigatorHooks(ctx, fs.HookTypeBeforeDownload, file); err != nil {
			m.l.Warning("Failed to execute navigator hooks: %s", err)
		}

		if useRedirect {
			reuseExisting := !m.user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionUniqueRedirectDirectLink))
			// Use redirect source
			link, err := fileClient.CreateDirectLink(ctx, file.ID(), file.Name(), m.user.Edges.Group.SpeedLimit, reuseExisting)
			if err != nil {
				ae.Add(url.String(), err)
				continue
			}

			linkHashID := hashid.EncodeSourceLinkID(m.hasher, link.ID)
			res = append(res, DirectLink{
				File: file,
				Url:  routes.MasterDirectLink(siteUrl, linkHashID, link.Name).String(),
			})
		} else {
			// Use direct source
			policy, d, err := m.getEntityPolicyDriver(ctx, target, nil)
			if err != nil {
				ae.Add(url.String(), err)
				continue
			}

			source := entitysource.NewEntitySource(target, d, policy, m.auth, m.settings, m.hasher, m.dep.RequestClient(),
				m.l, m.config, m.dep.MimeDetector(ctx), m.dep.EncryptorFactory(ctx))
			sourceUrl, err := source.Url(ctx,
				entitysource.WithSpeedLimit(int64(m.user.Edges.Group.SpeedLimit)),
				entitysource.WithDisplayName(file.Name()),
			)
			if err != nil {
				ae.Add(url.String(), err)
				continue
			}

			res = append(res, DirectLink{
				File: file,
				Url:  sourceUrl.Url,
			})
		}

	}

	return res, ae.Aggregate()
}

func (m *manager) GetUrlForRedirectedDirectLink(ctx context.Context, dl *ent.DirectLink, opts ...fs.Option) (string, *time.Time, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	file, err := m.fs.GetFileFromDirectLink(ctx, dl)
	if err != nil {
		return "", nil, err
	}

	// Find primary entity
	target, found := lo.Find(file.Entities(), func(entity fs.Entity) bool {
		return entity.ID() == file.PrimaryEntityID()
	})
	if !found {
		return "", nil, fs.ErrDirectLinkInvalid.WithError(fmt.Errorf("primary entity not found"))
	}
	primaryEntity := target

	// Generate url
	var (
		res    string
		expire *time.Time
	)

	// Try to read from cache.
	cacheKey := entityUrlCacheKey(primaryEntity.ID(), int64(dl.Speed), dl.Name, o.IsDownload,
		m.settings.SiteURL(ctx).String())
	if cached, ok := m.kv.Get(cacheKey); ok {
		cachedItem := cached.(EntityUrlCache)
		res = cachedItem.Url
		expire = cachedItem.ExpireAt
	} else {
		// Cache miss, Generate new url
		policy, d, err := m.getEntityPolicyDriver(ctx, primaryEntity, nil)
		if err != nil {
			return "", nil, err
		}

		source := entitysource.NewEntitySource(primaryEntity, d, policy, m.auth, m.settings, m.hasher, m.dep.RequestClient(),
			m.l, m.config, m.dep.MimeDetector(ctx), m.dep.EncryptorFactory(ctx))
		downloadUrl, err := source.Url(ctx,
			entitysource.WithExpire(o.Expire),
			entitysource.WithDownload(o.IsDownload),
			entitysource.WithSpeedLimit(int64(dl.Speed)),
			entitysource.WithDisplayName(dl.Name),
		)
		if err != nil {
			return "", nil, err
		}

		// Save into kv
		cacheValidDuration := expireTimeToTTL(o.Expire) - m.settings.EntityUrlCacheMargin(ctx)
		if cacheValidDuration > 0 {
			m.kv.Set(cacheKey, EntityUrlCache{
				Url:      downloadUrl.Url,
				ExpireAt: downloadUrl.ExpireAt,
			}, cacheValidDuration)
		}

		res = downloadUrl.Url
		expire = downloadUrl.ExpireAt
	}

	return res, expire, nil
}

func (m *manager) GetEntityUrls(ctx context.Context, args []GetEntityUrlArgs, opts ...fs.Option) ([]EntityUrl, *time.Time, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	var earliestExpireAt *time.Time
	res := make([]EntityUrl, len(args))
	ae := serializer.NewAggregateError()
	for i, arg := range args {
		file, err := m.fs.Get(
			ctx, arg.URI,
			dbfs.WithFileEntities(),
			dbfs.WithRequiredCapabilities(dbfs.NavigatorCapabilityDownloadFile),
		)
		if err != nil {
			ae.Add(arg.URI.String(), err)
			continue
		}

		if file.Type() != types.FileTypeFile {
			ae.Add(arg.URI.String(), fs.ErrEntityNotExist)
			continue
		}

		var (
			target fs.Entity
			found  bool
		)
		if arg.PreferredEntityID != "" {
			found, target = fs.FindDesiredEntity(file, arg.PreferredEntityID, m.hasher, nil)
			if !found {
				ae.Add(arg.URI.String(), fs.ErrEntityNotExist)
				continue
			}
		} else {
			// No preferred entity ID, use the primary version entity
			target = file.PrimaryEntity()
			if target == nil {
				ae.Add(arg.URI.String(), fs.ErrEntityNotExist)
				continue
			}
		}

		// Hooks for entity download
		if err := m.fs.ExecuteNavigatorHooks(ctx, fs.HookTypeBeforeDownload, file); err != nil {
			m.l.Warning("Failed to execute navigator hooks: %s", err)
		}

		policy, d, err := m.getEntityPolicyDriver(ctx, target, nil)
		if err != nil {
			ae.Add(arg.URI.String(), err)
			continue
		}

		// Try to read from cache.
		cacheKey := entityUrlCacheKey(target.ID(), o.DownloadSpeed, getEntityDisplayName(file, target), o.IsDownload,
			m.settings.SiteURL(ctx).String())
		if cached, ok := m.kv.Get(cacheKey); ok && !o.NoCache {
			cachedItem := cached.(EntityUrlCache)
			// Find the earliest expiry time
			if cachedItem.ExpireAt != nil && (earliestExpireAt == nil || cachedItem.ExpireAt.Before(*earliestExpireAt)) {
				earliestExpireAt = cachedItem.ExpireAt
			}
			res[i] = EntityUrl{
				Url:                        cachedItem.Url,
				BrowserDownloadDisplayName: cachedItem.BrowserDownloadDisplayName,
			}
			continue
		}

		// Cache miss, Generate new url
		source := entitysource.NewEntitySource(target, d, policy, m.auth, m.settings, m.hasher, m.dep.RequestClient(),
			m.l, m.config, m.dep.MimeDetector(ctx), m.dep.EncryptorFactory(ctx))
		downloadUrl, err := source.Url(ctx,
			entitysource.WithExpire(o.Expire),
			entitysource.WithDownload(o.IsDownload),
			entitysource.WithSpeedLimit(o.DownloadSpeed),
			entitysource.WithDisplayName(getEntityDisplayName(file, target)),
		)
		if err != nil {
			ae.Add(arg.URI.String(), err)
			continue
		}

		// Find the earliest expiry time
		if downloadUrl.ExpireAt != nil && (earliestExpireAt == nil || downloadUrl.ExpireAt.Before(*earliestExpireAt)) {
			earliestExpireAt = downloadUrl.ExpireAt
		}

		// Save into kv
		cacheValidDuration := expireTimeToTTL(o.Expire) - m.settings.EntityUrlCacheMargin(ctx)
		if cacheValidDuration > 0 {
			m.kv.Set(cacheKey, EntityUrlCache{
				Url:      downloadUrl.Url,
				ExpireAt: downloadUrl.ExpireAt,
			}, cacheValidDuration)
		}

		res[i] = EntityUrl{
			Url: downloadUrl.Url,
		}
		if d.Capabilities().BrowserRelayedDownload {
			res[i].BrowserDownloadDisplayName = getEntityDisplayName(file, target)
		}
	}

	return res, earliestExpireAt, ae.Aggregate()
}

func (m *manager) GetEntitySource(ctx context.Context, entityID int, opts ...fs.Option) (entitysource.EntitySource, error) {
	o := newOption()
	for _, opt := range opts {
		opt.Apply(o)
	}

	var (
		entity fs.Entity
		err    error
	)

	if o.Entity != nil {
		entity = o.Entity
	} else {
		entity, err = m.fs.GetEntity(ctx, entityID)
		if err != nil {
			return nil, err
		}

		if entity.ReferenceCount() == 0 {
			return nil, fs.ErrEntityNotExist
		}
	}

	policy, handler, err := m.getEntityPolicyDriver(ctx, entity, o.Policy)
	if err != nil {
		return nil, err
	}

	return entitysource.NewEntitySource(entity, handler, policy, m.auth, m.settings, m.hasher, m.dep.RequestClient(), m.l,
		m.config, m.dep.MimeDetector(ctx), m.dep.EncryptorFactory(ctx), entitysource.WithContext(ctx), entitysource.WithThumb(o.IsThumb)), nil
}

func (l *manager) SetCurrentVersion(ctx context.Context, path *fs.URI, version int) error {
	return l.fs.VersionControl(ctx, path, version, false)
}

func (l *manager) DeleteVersion(ctx context.Context, path *fs.URI, version int) error {
	return l.fs.VersionControl(ctx, path, version, true)
}

func (l *manager) ListPhysical(ctx context.Context, path string, policyID int, recursive bool, progress driver.ListProgressFunc) ([]fs.PhysicalObject, error) {
	policy, err := l.dep.StoragePolicyClient().GetPolicyByID(ctx, policyID)
	if err != nil {
		return nil, err
	}

	driver, err := l.GetStorageDriver(ctx, policy)
	if err != nil {
		return nil, err
	}

	return driver.List(ctx, path, progress, recursive)
}

func (l *manager) ImportPhysical(ctx context.Context, dst *fs.URI, policyId int, src fs.PhysicalObject, completeHook bool) error {
	targetUri := dst.Join(src.RelativePath)
	req := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:                    targetUri,
			UploadSessionID:        uuid.Must(uuid.NewV4()).String(),
			Size:                   src.Size,
			PreferredStoragePolicy: policyId,
			SavePath:               src.Source,
			LastModified:           &src.LastModify,
		},
		ImportFrom: &src,
	}

	// Prepare for upload
	uploadSession, err := l.fs.PrepareUpload(ctx, req)
	if err != nil {
		return fmt.Errorf("faield to prepare uplaod: %w", err)
	}
	if completeHook {
		d, err := l.GetStorageDriver(ctx, l.CastStoragePolicyOnSlave(ctx, uploadSession.Policy))
		if err != nil {
			return err
		}

		l.onNewEntityUploaded(ctx, uploadSession, d)
	}

	return nil
}

func entityUrlCacheKey(id int, speed int64, displayName string, download bool, siteUrl string) string {
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("%d_%d_%s_%t_%s", id,
		speed, displayName, download, siteUrl)))
	hashRes := hex.EncodeToString(hash.Sum(nil))

	return fmt.Sprintf("%s_%s", EntityUrlCacheKeyPrefix, hashRes)
}
