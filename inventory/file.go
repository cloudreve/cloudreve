package inventory

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/directlink"
	"github.com/cloudreve/Cloudreve/v4/ent/entity"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/ent/metadata"
	"github.com/cloudreve/Cloudreve/v4/ent/predicate"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/ent/share"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
	"golang.org/x/tools/container/intsets"
)

const (
	RootFolderName = ""
	SearchWildcard = "*"
	MaxMetadataLen = 65535
)

type (
	// Ctx keys for eager loading options.
	LoadFileEntity          struct{}
	LoadFileMetadata        struct{}
	LoadFilePublicMetadata  struct{}
	LoadFileShare           struct{}
	LoadFileUser            struct{}
	LoadFileDirectLink      struct{}
	LoadEntityUser          struct{}
	LoadEntityStoragePolicy struct{}
	LoadEntityFile          struct{}

	// Parameters for file list
	ListFileParameters struct {
		*PaginationArgs
		// Whether to mix folder with files in results, only applied to cursor pagination
		MixedType bool
		// Whether to include only folder in results, only applied to cursor pagination
		FolderOnly bool
		// SharedWithMe indicates whether to list files shared with the user
		SharedWithMe bool
		Search       *SearchFileParameters
	}

	FlattenListFileParameters struct {
		*PaginationArgs
		UserID          int
		Name            string
		StoragePolicyID int
	}

	SearchFileParameters struct {
		Name []string
		// NameOperatorOr is true if the name should match any of the given names, false if all of them
		NameOperatorOr bool
		Metadata       map[string]string
		Type           *types.FileType
		UseFullText    bool
		CaseFolding    bool
		Category       string
		SizeGte        int64
		SizeLte        int64
		CreatedAtGte   *time.Time
		CreatedAtLte   *time.Time
		UpdatedAtGte   *time.Time
		UpdatedAtLte   *time.Time
	}

	ListEntityParameters struct {
		*PaginationArgs
		EntityType      *types.EntityType
		UserID          int
		StoragePolicyID int
	}

	ListEntityResult struct {
		Entities []*ent.Entity
		*PaginationResults
	}

	ListFileResult struct {
		Files     []*ent.File
		MixedType bool
		*PaginationResults
	}

	CreateFileParameters struct {
		FileType            types.FileType
		Name                string
		Metadata            map[string]string
		MetadataPrivateMask map[string]bool
		IsSymbolic          bool
		StoragePolicyID     int
		*EntityParameters
	}

	CreateFolderParameters struct {
		Owner               int
		Name                string
		IsSymbolic          bool
		Metadata            map[string]string
		MetadataPrivateMask map[string]bool
	}

	EntityParameters struct {
		ModifiedAt      *time.Time
		OwnerID         int
		EntityType      types.EntityType
		StoragePolicyID int
		Source          string
		Size            int64
		UploadSessionID uuid.UUID
		Importing       bool
	}

	RelocateEntityParameter struct {
		Entity                   *ent.Entity
		NewSource                string
		ParentFiles              []int
		PrimaryEntityParentFiles []int
	}
)

type FileClient interface {
	TxOperator
	// GetByHashID returns a file by its hashID.
	GetByHashID(ctx context.Context, hashID string) (*ent.File, error)
	// GetParentFile returns the parent folder of a given file
	GetParentFile(ctx context.Context, root *ent.File, eagerLoading bool) (*ent.File, error)
	// Search file by name from a given root. eagerLoading indicates whether to load edges determined by ctx.
	GetChildFile(ctx context.Context, root *ent.File, ownerID int, child string, eagerLoading bool) (*ent.File, error)
	// Get all files under a given root
	GetChildFiles(ctx context.Context, args *ListFileParameters, ownerID int, roots ...*ent.File) (*ListFileResult, error)
	// Root returns the root folder of a given user
	Root(ctx context.Context, user *ent.User) (*ent.File, error)
	// CreateOrGetFolder creates a folder with given name under root, or return the existed one
	CreateFolder(ctx context.Context, root *ent.File, args *CreateFolderParameters) (*ent.File, error)
	// GetByIDs returns files with given IDs. The result is paginated. Caller should keep calling this until
	// returned next page is negative.
	GetByIDs(ctx context.Context, ids []int, page int) ([]*ent.File, int, error)
	// GetByID returns a file by its ID.
	GetByID(ctx context.Context, ids int) (*ent.File, error)
	// GetEntitiesByIDs returns entities with given IDs. The result is paginated. Caller should keep calling this until
	// returned next page is negative.
	GetEntitiesByIDs(ctx context.Context, ids []int, page int) ([]*ent.Entity, int, error)
	// GetEntityByID returns an entity by its ID.
	GetEntityByID(ctx context.Context, id int) (*ent.Entity, error)
	// Rename renames a file
	Rename(ctx context.Context, original *ent.File, newName string) (*ent.File, error)
	// SetParent sets parent of group of files
	SetParent(ctx context.Context, files []*ent.File, parent *ent.File) error
	// CreateFile creates a file with given parameters, returns created File, default Entity, and storage diff for owner.
	CreateFile(ctx context.Context, root *ent.File, args *CreateFileParameters) (*ent.File, *ent.Entity, StorageDiff, error)
	// UpgradePlaceholder upgrades a placeholder entity to a real version entity
	UpgradePlaceholder(ctx context.Context, file *ent.File, modifiedAt *time.Time, entityId int, entityType types.EntityType) error
	// RemoveMetadata removes metadata from a file
	RemoveMetadata(ctx context.Context, file *ent.File, keys ...string) error
	// CreateEntity creates an entity with given parameters, returns created Entity, and storage diff for owner.
	CreateEntity(ctx context.Context, file *ent.File, args *EntityParameters) (*ent.Entity, StorageDiff, error)
	// RemoveStaleEntities hard-delete stale placeholder entities
	RemoveStaleEntities(ctx context.Context, file *ent.File) (StorageDiff, error)
	// RemoveEntitiesByID hard-delete entities by IDs.
	RemoveEntitiesByID(ctx context.Context, ids ...int) (map[int]int64, error)
	// CapEntities caps the number of entities of a given file. The oldest entities will be unlinked
	// if entity count exceed limit.
	CapEntities(ctx context.Context, file *ent.File, owner *ent.User, max int, entityType types.EntityType) (StorageDiff, error)
	// UpsertMetadata update or insert metadata
	UpsertMetadata(ctx context.Context, file *ent.File, data map[string]string, privateMask map[string]bool) error
	// Copy copies a layer of file to its corresponding destination folder. dstMap is a map from src parent ID to dst parent Files.
	Copy(ctx context.Context, files []*ent.File, dstMap map[int][]*ent.File) (map[int][]*ent.File, StorageDiff, error)
	// Delete deletes a group of files (and related models) with given entity recycle option
	Delete(ctx context.Context, files []*ent.File, options *types.EntityRecycleOption) ([]*ent.Entity, StorageDiff, error)
	// StaleEntities returns stale entities of a given file. If ID is not provided, all entities
	// will be examined.
	StaleEntities(ctx context.Context, ids ...int) ([]*ent.Entity, error)
	// QueryMetadata load metadata of a given file
	QueryMetadata(ctx context.Context, root *ent.File) error
	// SoftDelete soft-deletes a file, also renaming it to a random name
	SoftDelete(ctx context.Context, file *ent.File) error
	// SetPrimaryEntity sets primary entity of a file
	SetPrimaryEntity(ctx context.Context, file *ent.File, entityID int) error
	// UnlinkEntity unlinks an entity from a file
	UnlinkEntity(ctx context.Context, entity *ent.Entity, file *ent.File, owner *ent.User) (StorageDiff, error)
	// CreateDirectLink creates a direct link for a file
	CreateDirectLink(ctx context.Context, fileID int, name string, speed int, reuse bool) (*ent.DirectLink, error)
	// CountByTimeRange counts files created in a given time range
	CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error)
	// CountEntityByTimeRange counts entities created in a given time range
	CountEntityByTimeRange(ctx context.Context, start, end *time.Time) (int, error)
	// CountEntityByStoragePolicyID counts entities by storage policy ID
	CountEntityByStoragePolicyID(ctx context.Context, storagePolicyID int) (int, int, error)
	// IsStoragePolicyUsedByEntities checks if a storage policy is used by entities
	IsStoragePolicyUsedByEntities(ctx context.Context, policyID int) (bool, error)
	// DeleteByUser deletes all files by a given user
	DeleteByUser(ctx context.Context, uid int) error
	// FlattenListFiles list files ignoring hierarchy
	FlattenListFiles(ctx context.Context, args *FlattenListFileParameters) (*ListFileResult, error)
	// Update updates a file
	Update(ctx context.Context, file *ent.File) (*ent.File, error)
	// ListEntities lists entities
	ListEntities(ctx context.Context, args *ListEntityParameters) (*ListEntityResult, error)
	// UpdateProps updates props of a file
	UpdateProps(ctx context.Context, file *ent.File, props *types.FileProps) (*ent.File, error)
}

func NewFileClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) FileClient {
	return &fileClient{client: client, maxSQlParam: sqlParamLimit(dbType), hasher: hasher}
}

type fileClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *fileClient) SetClient(newClient *ent.Client) TxOperator {
	return &fileClient{client: newClient, maxSQlParam: c.maxSQlParam, hasher: c.hasher}
}

func (c *fileClient) GetClient() *ent.Client {
	return c.client
}

func (f *fileClient) Update(ctx context.Context, file *ent.File) (*ent.File, error) {
	q := f.client.File.UpdateOne(file).
		SetName(file.Name).
		SetStoragePoliciesID(file.StoragePolicyFiles)

	existingMetadata, err := f.client.Metadata.Query().Where(metadata.FileID(file.ID)).All(ctx)
	if err != nil {
		return nil, err
	}

	metadataIDs := lo.Map(file.Edges.Metadata, func(item *ent.Metadata, index int) int {
		return item.ID
	})

	existingMetadataIds := lo.Map(existingMetadata, func(item *ent.Metadata, index int) int {
		return item.ID
	})

	metadataToDelete := lo.Without(existingMetadataIds, metadataIDs...)

	// process metadata diff, delete metadata not in file.Edges.Metadata
	_, err = f.client.Metadata.Delete().Where(metadata.FileID(file.ID), metadata.IDIn(metadataToDelete...)).Exec(schema.SkipSoftDelete(ctx))
	if err != nil {
		return nil, err
	}

	// Update presented metadata
	for _, metadata := range file.Edges.Metadata {
		f.client.Metadata.UpdateOne(metadata).
			SetName(metadata.Name).
			SetValue(metadata.Value).
			SetIsPublic(metadata.IsPublic).
			Save(ctx)
	}

	// process direct link diff, delete direct link not in file.Edges.DirectLinks
	_, err = f.client.DirectLink.Delete().Where(directlink.FileID(file.ID), directlink.IDNotIn(lo.Map(file.Edges.DirectLinks, func(item *ent.DirectLink, index int) int {
		return item.ID
	})...)).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return q.Save(ctx)
}

func (f *fileClient) UpdateProps(ctx context.Context, file *ent.File, props *types.FileProps) (*ent.File, error) {
	file, err := f.client.File.UpdateOne(file).
		SetProps(props).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (f *fileClient) CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error) {
	if start == nil || end == nil {
		return f.client.File.Query().Count(ctx)
	}

	return f.client.File.Query().Where(file.CreatedAtGTE(*start), file.CreatedAtLT(*end)).Count(ctx)
}

func (f *fileClient) CountEntityByTimeRange(ctx context.Context, start, end *time.Time) (int, error) {
	if start == nil || end == nil {
		return f.client.Entity.Query().Count(ctx)
	}

	return f.client.Entity.Query().Where(entity.CreatedAtGTE(*start), entity.CreatedAtLT(*end)).Count(ctx)
}

func (f *fileClient) CountEntityByStoragePolicyID(ctx context.Context, storagePolicyID int) (int, int, error) {
	var v []struct {
		Sum   int `json:"sum"`
		Count int `json:"count"`
	}

	err := f.client.Entity.Query().Where(entity.StoragePolicyEntities(storagePolicyID)).
		Aggregate(
			ent.Sum(entity.FieldSize),
			ent.Count(),
		).Scan(ctx, &v)
	if err != nil {
		return 0, 0, err
	}

	return v[0].Count, v[0].Sum, nil
}

func (f *fileClient) CreateDirectLink(ctx context.Context, file int, name string, speed int, reuse bool) (*ent.DirectLink, error) {
	if reuse {
		// Find existed
		existed, err := f.client.DirectLink.
			Query().
			Where(directlink.FileID(file), directlink.Name(name), directlink.Speed(speed)).First(ctx)
		if err == nil {
			return existed, nil
		}
	}

	return f.client.DirectLink.
		Create().
		SetFileID(file).
		SetName(name).
		SetSpeed(speed).
		SetDownloads(0).
		Save(ctx)
}

func (f *fileClient) GetByHashID(ctx context.Context, hashID string) (*ent.File, error) {
	id, err := f.hasher.Decode(hashID, hashid.FileID)
	if err != nil {
		return nil, fmt.Errorf("ailed to decode hash id %q: %w", hashID, err)
	}

	return withFileEagerLoading(ctx, f.client.File.Query().Where(file.ID(id))).First(ctx)
}

func (f *fileClient) SoftDelete(ctx context.Context, file *ent.File) error {
	newName := uuid.Must(uuid.NewV4())
	// Rename file to random UUID and make it stale
	_, err := f.client.File.UpdateOne(file).
		SetName(newName.String()).
		ClearParent().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to soft delete file %d: %w", file.ID, err)
	}

	return err
}

func (f *fileClient) RemoveEntitiesByID(ctx context.Context, ids ...int) (map[int]int64, error) {
	groups, _ := f.batchInConditionEntityID(intsets.MaxInt, 10, 1, ids)
	// storageReduced stores the relation between owner ID and storage reduced.
	storageReduced := make(map[int]int64)

	ctx = schema.SkipSoftDelete(ctx)
	for _, group := range groups {
		// For entities that are referenced by files, we need to reduce the storage of the owner of the files.
		entities, err := f.client.Entity.Query().
			Where(group).
			Where(entity.ReferenceCountGT(0)).
			WithFile().
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query entities %v: %w", group, err)
		}

		for _, entity := range entities {
			if entity.Edges.File == nil {
				continue
			}

			for _, file := range entity.Edges.File {
				storageReduced[file.OwnerID] -= entity.Size
			}
		}

		_, err = f.client.Entity.Delete().
			Where(group).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query entities %v: %w", group, err)
		}
	}

	return storageReduced, nil
}

func (f *fileClient) StaleEntities(ctx context.Context, ids ...int) ([]*ent.Entity, error) {
	res := make([]*ent.Entity, 0, len(ids))
	if len(ids) > 0 {
		// If explicit IDs are given, we can query them directly
		groups, _ := f.batchInConditionEntityID(intsets.MaxInt, 10, 1, ids)
		for _, group := range groups {
			entities, err := f.client.Entity.Query().
				Where(group).
				All(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to query entities %v: %w", group, err)
			}
			res = append(res, entities...)
		}

		return res, nil
	}

	// No explicit IDs are given, we need to query all entities
	entities, err := f.client.Entity.Query().
		Where(entity.Or(
			entity.ReferenceCountLTE(0),
		)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query stale entities: %w", err)
	}

	return entities, nil
}

func (f *fileClient) DeleteByUser(ctx context.Context, uid int) error {
	batchSize := capPageSize(f.maxSQlParam, intsets.MaxInt, 10)
	for {
		files, err := f.client.File.Query().
			WithEntities().
			Where(file.OwnerID(uid)).
			Limit(batchSize).
			All(ctx)
		if err != nil {
			return fmt.Errorf("failed to query files: %w", err)
		}

		if len(files) == 0 {
			break
		}

		if _, _, err := f.Delete(ctx, files, nil); err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
	}

	return nil
}

func (f *fileClient) Delete(ctx context.Context, files []*ent.File, options *types.EntityRecycleOption) ([]*ent.Entity, StorageDiff, error) {
	// 1. Decrease reference count for all entities;
	// entities stores the relation between its reference count in `files` and entity ID.
	entities := make(map[int]int)
	// storageReduced stores the relation between owner ID and storage reduced.
	storageReduced := make(map[int]int64)
	for _, fi := range files {
		fileEntities, err := fi.Edges.EntitiesOrErr()
		if err != nil {
			return nil, nil, err
		}

		for _, e := range fileEntities {
			entities[e.ID]++
			storageReduced[fi.OwnerID] -= e.Size
		}
	}

	// Group entities by their reference count.
	uniqueEntities := lo.Keys(entities)
	entitiesGrouped := lo.GroupBy(uniqueEntities, func(e int) int {
		return entities[e]
	})

	for ref, entityGroup := range entitiesGrouped {
		entityPageGroup, _ := f.batchInConditionEntityID(intsets.MaxInt, 10, 1, entityGroup)
		for _, group := range entityPageGroup {
			if err := f.client.Entity.Update().
				Where(group).
				AddReferenceCount(-1 * ref).
				Exec(ctx); err != nil {
				return nil, nil, fmt.Errorf("failed to decrease reference count for entities %v: %w", group, err)
			}
		}
	}

	// 2. Filter out entities with <=0 reference count, Update recycle options for above entities;
	entityGroup, _ := f.batchInConditionEntityID(intsets.MaxInt, 10, 1, uniqueEntities)
	toBeRecycled := make([]*ent.Entity, 0, len(entities))
	for _, group := range entityGroup {
		e, err := f.client.Entity.Query().Where(group).Where(entity.ReferenceCountLTE(0)).All(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query orphan entities %v: %w", group, err)
		}

		toBeRecycled = append(toBeRecycled, e...)
	}

	// 3. Update recycle options for above entities;
	pageSize := capPageSize(f.maxSQlParam, intsets.MaxInt, 10)
	chunks := lo.Chunk(lo.Map(toBeRecycled, func(item *ent.Entity, index int) int {
		return item.ID
	}), max(pageSize, 1))
	for _, chunk := range chunks {
		if err := f.client.Entity.Update().
			Where(entity.IDIn(chunk...)).
			SetRecycleOptions(options).
			Exec(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to update recycle options for entities %v: %w", chunk, err)
		}
	}

	hardDeleteCtx := schema.SkipSoftDelete(ctx)
	fileGroups, chunks := f.batchInCondition(intsets.MaxInt, 10, 1,
		lo.Map(files, func(file *ent.File, index int) int {
			return file.ID
		}),
	)

	for i, group := range fileGroups {
		// 4. Delete shares/metadata/directlinks if needed;
		if _, err := f.client.Share.Delete().Where(share.HasFileWith(group)).Exec(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to delete shares of files %v: %w", group, err)
		}

		if _, err := f.client.Metadata.Delete().Where(metadata.FileIDIn(chunks[i]...)).Exec(schema.SkipSoftDelete(ctx)); err != nil {
			return nil, nil, fmt.Errorf("failed to delete metadata of files %v: %w", group, err)
		}

		if _, err := f.client.DirectLink.Delete().Where(directlink.FileIDIn(chunks[i]...)).Exec(hardDeleteCtx); err != nil {
			return nil, nil, fmt.Errorf("failed to delete direct links of files %v: %w", group, err)
		}

		// 5. Delete files.
		if _, err := f.client.File.Delete().Where(group).Exec(hardDeleteCtx); err != nil {
			return nil, nil, fmt.Errorf("failed to delete files %v: %w", group, err)
		}
	}

	return toBeRecycled, storageReduced, nil
}

func (f *fileClient) Copy(ctx context.Context, files []*ent.File, dstMap map[int][]*ent.File) (map[int][]*ent.File, StorageDiff, error) {
	pageSize := capPageSize(f.maxSQlParam, intsets.MaxInt, 10)
	// 1. Copy files and metadata
	copyFileStm := lo.Map(files, func(file *ent.File, index int) *ent.FileCreate {

		stm := f.client.File.Create().
			SetName(file.Name).
			SetOwnerID(dstMap[file.FileChildren][0].OwnerID).
			SetSize(file.Size).
			SetType(file.Type).
			SetParent(dstMap[file.FileChildren][0]).
			SetIsSymbolic(file.IsSymbolic)
		if file.StoragePolicyFiles > 0 {
			stm.SetStoragePolicyFiles(file.StoragePolicyFiles)
		}
		if file.PrimaryEntity > 0 {
			stm.SetPrimaryEntity(file.PrimaryEntity)
		}

		if file.Props != nil && dstMap[file.FileChildren][0].OwnerID == file.OwnerID {
			stm.SetProps(file.Props)
		}

		return stm
	})

	metadataStm := []*ent.MetadataCreate{}
	entityStm := []*ent.EntityUpdate{}
	newDstMap := make(map[int][]*ent.File, len(files))
	sizeDiff := int64(0)
	for index, stm := range copyFileStm {
		newFile, err := stm.Save(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to copy file: %w", err)
		}

		fileMetadata, err := files[index].Edges.MetadataOrErr()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get metadata of file: %w", err)
		}

		metadataStm = append(metadataStm, lo.Map(fileMetadata, func(metadata *ent.Metadata, index int) *ent.MetadataCreate {
			return f.client.Metadata.Create().
				SetName(metadata.Name).
				SetValue(metadata.Value).
				SetFile(newFile).
				SetIsPublic(metadata.IsPublic)
		})...)

		fileEntities, err := files[index].Edges.EntitiesOrErr()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get entities of file: %w", err)
		}
		ids := lo.FilterMap(fileEntities, func(entity *ent.Entity, index int) (int, bool) {
			// Skip entities that are still uploading
			if entity.UploadSessionID != nil {
				return 0, false
			}
			sizeDiff += entity.Size
			return entity.ID, true
		})

		entityBatch, _ := f.batchInConditionEntityID(intsets.MaxInt, 10, 1, ids)
		entityStm = append(entityStm, lo.Map(entityBatch, func(batch predicate.Entity, index int) *ent.EntityUpdate {
			return f.client.Entity.Update().Where(batch).AddReferenceCount(1).AddFile(newFile)
		})...)

		newAncestorChain := append(newDstMap[files[index].ID], newFile)
		newAncestorChain[0], newAncestorChain[len(newAncestorChain)-1] = newAncestorChain[len(newAncestorChain)-1], newAncestorChain[0]
		newDstMap[files[index].ID] = newAncestorChain
	}

	// 2. Copy metadata by group
	chunkedMetadataStm := lo.Chunk(metadataStm, pageSize/10)
	for _, chunk := range chunkedMetadataStm {
		if err := f.client.Metadata.CreateBulk(chunk...).Exec(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to copy metadata: %w", err)
		}
	}

	// 3. Copy entity relations
	for _, stm := range entityStm {
		if err := stm.Exec(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to copy entity: %w", err)
		}
	}

	return newDstMap, map[int]int64{dstMap[files[0].FileChildren][0].OwnerID: sizeDiff}, nil
}

func (f *fileClient) UpsertMetadata(ctx context.Context, file *ent.File, data map[string]string, privateMask map[string]bool) error {
	// Validate value length
	for key, value := range data {
		if len(value) > MaxMetadataLen {
			return fmt.Errorf("metadata value of key %s is too long", key)
		}
	}

	if err := f.client.Metadata.
		CreateBulk(lo.MapToSlice(data, func(key string, value string) *ent.MetadataCreate {
			isPrivate := false
			if privateMask != nil {
				_, isPrivate = privateMask[key]
			}

			return f.client.Metadata.Create().
				SetName(key).
				SetValue(value).
				SetFile(file).
				SetIsPublic(!isPrivate).
				SetNillableDeletedAt(nil)
		})...).
		OnConflictColumns(metadata.FieldFileID, metadata.FieldName).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}

	return nil
}

func (f *fileClient) RemoveMetadata(ctx context.Context, file *ent.File, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	ctx = schema.SkipSoftDelete(ctx)
	groups, _ := f.batchInConditionMetadataName(intsets.MaxInt, 10, 1, keys)
	for _, group := range groups {
		if _, err := f.client.Metadata.Delete().Where(metadata.FileID(file.ID), group).Exec(ctx); err != nil {
			return fmt.Errorf("failed to remove metadata: %v", err)
		}
	}

	return nil
}

func (f *fileClient) UpgradePlaceholder(ctx context.Context, file *ent.File, modifiedAt *time.Time, entityId int,
	entityType types.EntityType) error {
	entities, err := file.Edges.EntitiesOrErr()
	if err != nil {
		return err
	}

	placeholder, found := lo.Find(entities, func(e *ent.Entity) bool {
		return e.ID == entityId
	})
	if !found {
		return fmt.Errorf("no identity with id %d entity for file %d", entityId, file.ID)
	}

	stm := f.client.Entity.
		UpdateOne(placeholder).
		ClearUploadSessionID()
	if modifiedAt != nil {
		stm.SetUpdatedAt(*modifiedAt)
	}

	if err := stm.Exec(ctx); err != nil {
		return fmt.Errorf("failed to upgrade placeholder: %v", err)
	}

	if entityType == types.EntityTypeVersion {
		if err := f.client.File.UpdateOne(file).
			SetSize(placeholder.Size).
			SetPrimaryEntity(placeholder.ID).
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to upgrade file primary entity: %v", err)
		}
	}
	return nil
}

func (f *fileClient) SetPrimaryEntity(ctx context.Context, file *ent.File, entityID int) error {
	return f.client.File.UpdateOne(file).SetPrimaryEntity(entityID).Exec(ctx)
}

func (f *fileClient) CreateFile(ctx context.Context, root *ent.File, args *CreateFileParameters) (*ent.File, *ent.Entity, StorageDiff, error) {
	var defaultEntity *ent.Entity
	stm := f.client.File.
		Create().
		SetOwnerID(root.OwnerID).
		SetType(int(args.FileType)).
		SetName(args.Name).
		SetParent(root).
		SetIsSymbolic(args.IsSymbolic).
		SetStoragePoliciesID(args.StoragePolicyID)

	if args.EntityParameters != nil && args.EntityParameters.Importing {
		stm.SetSize(args.EntityParameters.Size)
	}

	newFile, err := stm.Save(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create file: %v", err)
	}

	// Create default primary file entity if needed
	var storageDiff StorageDiff
	if args.EntityParameters != nil {
		args.EntityParameters.OwnerID = root.OwnerID
		args.EntityParameters.StoragePolicyID = args.StoragePolicyID
		defaultEntity, storageDiff, err = f.CreateEntity(ctx, newFile, args.EntityParameters)
		if err != nil {
			return nil, nil, storageDiff, fmt.Errorf("failed to create default entity: %v", err)
		}

		if args.EntityParameters.Importing {
			if err := f.client.File.UpdateOne(newFile).SetPrimaryEntity(defaultEntity.ID).Exec(ctx); err != nil {
				return nil, nil, storageDiff, fmt.Errorf("failed to set primary entity: %v", err)
			}
		}
	}

	// Create metadata if needed
	if len(args.Metadata) > 0 {
		_, err := f.client.Metadata.
			CreateBulk(lo.MapToSlice(args.Metadata, func(key, value string) *ent.MetadataCreate {
				_, isPrivate := args.MetadataPrivateMask[key]
				return f.client.Metadata.Create().
					SetName(key).
					SetValue(value).
					SetFile(newFile).
					SetIsPublic(!isPrivate)
			})...).
			Save(ctx)
		if err != nil {
			return nil, nil, storageDiff, fmt.Errorf("failed to create metadata: %v", err)
		}
	}

	return newFile, defaultEntity, storageDiff, err

}

func (f *fileClient) CapEntities(ctx context.Context, file *ent.File, owner *ent.User, max int, entityType types.EntityType) (StorageDiff, error) {
	entities, err := file.Edges.EntitiesOrErr()
	if err != nil {
		return nil, fmt.Errorf("failed to cap file entities: %v", err)
	}

	versionCount := 0
	diff := make(StorageDiff)
	for _, e := range entities {
		if e.Type != int(entityType) {
			continue
		}

		versionCount++
		if versionCount > max {
			// By default, eager-loaded entity is sorted by ID in descending order.
			// So we can just unlink the entity and it will be the older version.
			newDiff, err := f.UnlinkEntity(ctx, e, file, owner)
			if err != nil {
				return diff, fmt.Errorf("failed to cap file entities: %v", err)
			}

			diff.Merge(newDiff)
		}
	}

	return diff, nil
}

func (f *fileClient) UnlinkEntity(ctx context.Context, entity *ent.Entity, file *ent.File, owner *ent.User) (StorageDiff, error) {
	if err := f.client.Entity.UpdateOne(entity).RemoveFile(file).AddReferenceCount(-1).Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to unlink entity: %v", err)
	}

	return map[int]int64{owner.ID: entity.Size * int64(-1)}, nil
}

func (f *fileClient) IsStoragePolicyUsedByEntities(ctx context.Context, policyID int) (bool, error) {
	res, err := f.client.Entity.Query().Where(entity.StoragePolicyEntities(policyID)).Limit(1).All(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if storage policy is used by entities: %v", err)
	}

	if len(res) > 0 {
		return true, nil
	}

	return false, nil
}

func (f *fileClient) RemoveStaleEntities(ctx context.Context, file *ent.File) (StorageDiff, error) {
	entities, err := file.Edges.EntitiesOrErr()
	if err != nil {
		return nil, fmt.Errorf("failed to get stale entities: %v", err)
	}

	sizeReduced := int64(0)
	ids := lo.FilterMap(entities, func(e *ent.Entity, index int) (int, bool) {
		if e.ReferenceCount == 1 && e.UploadSessionID != nil {
			sizeReduced += e.Size
			return e.ID, true
		}

		return 0, false
	})

	if len(ids) > 0 {
		if err = f.client.Entity.Update().
			Where(entity.IDIn(ids...)).
			RemoveFile(file).
			AddReferenceCount(-1).Exec(ctx); err != nil {
			return nil, fmt.Errorf("failed to remove stale entities: %v", err)
		}
	}

	return map[int]int64{file.OwnerID: sizeReduced * -1}, nil
}

func (f *fileClient) CreateEntity(ctx context.Context, file *ent.File, args *EntityParameters) (*ent.Entity, StorageDiff, error) {
	createdBy := UserFromContext(ctx)
	stm := f.client.Entity.
		Create().
		SetType(int(args.EntityType)).
		SetSource(args.Source).
		SetSize(args.Size).
		SetStoragePolicyID(args.StoragePolicyID)

	if createdBy != nil && !IsAnonymousUser(createdBy) {
		stm.SetUser(createdBy)
	}

	if args.ModifiedAt != nil {
		stm.SetUpdatedAt(*args.ModifiedAt)
	}

	if args.UploadSessionID != uuid.Nil && !args.Importing {
		stm.SetUploadSessionID(args.UploadSessionID)
	}

	created, err := stm.Save(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create file entity: %v", err)
	}

	diff := map[int]int64{file.OwnerID: created.Size}

	if err := f.client.File.UpdateOne(file).AddEntities(created).Exec(ctx); err != nil {
		return nil, diff, fmt.Errorf("failed to add file entity: %v", err)
	}

	return created, diff, nil
}

func (f *fileClient) SetParent(ctx context.Context, files []*ent.File, parent *ent.File) error {
	groups, _ := f.batchInCondition(intsets.MaxInt, 10, 1, lo.Map(files, func(file *ent.File, index int) int {
		return file.ID
	}))
	for _, group := range groups {
		_, err := f.client.File.Update().SetParent(parent).Where(group).Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to set parent field: %w", err)
		}
	}

	return nil
}

func (f *fileClient) GetParentFile(ctx context.Context, root *ent.File, eagerLoading bool) (*ent.File, error) {
	query := f.client.File.QueryParent(root)
	if eagerLoading {
		query = withFileEagerLoading(ctx, query)
	}

	return query.First(ctx)
}

func (f *fileClient) QueryMetadata(ctx context.Context, root *ent.File) error {
	metadata, err := f.client.File.QueryMetadata(root).All(ctx)
	if err != nil {
		return err
	}
	root.SetMetadata(metadata)
	return nil
}

func (f *fileClient) GetChildFile(ctx context.Context, root *ent.File, ownerID int, child string, eagerLoading bool) (*ent.File, error) {
	query := f.childFileQuery(ownerID, false, root)
	if eagerLoading {
		query = withFileEagerLoading(ctx, query)
	}

	return query.
		Where(file.Name(child)).First(ctx)
}

func (f *fileClient) GetChildFiles(ctx context.Context, args *ListFileParameters, ownerID int, roots ...*ent.File) (*ListFileResult, error) {
	rawQuery := f.childFileQuery(ownerID, args.SharedWithMe, roots...)
	query := withFileEagerLoading(ctx, rawQuery)
	if args.Search != nil {
		query = f.searchQuery(query, args.Search, roots, ownerID)
	}

	var (
		files         []*ent.File
		err           error
		paginationRes *PaginationResults
	)

	if args.UseCursorPagination || args.Search != nil {
		files, paginationRes, err = f.cursorPagination(ctx, query, args, 10)
	} else {
		files, paginationRes, err = f.offsetPagination(ctx, query, args, 10)
	}

	if err != nil {
		return nil, fmt.Errorf("query failed with paginiation: %w", err)
	}

	return &ListFileResult{
		Files:             files,
		PaginationResults: paginationRes,
		MixedType:         args.MixedType,
	}, nil
}

func (f *fileClient) Root(ctx context.Context, user *ent.User) (*ent.File, error) {
	return f.client.User.
		QueryFiles(user).
		Where(file.Not(file.HasParent())).
		Where(file.Name(RootFolderName)).
		First(ctx)
}

func (f *fileClient) CreateFolder(ctx context.Context, root *ent.File, args *CreateFolderParameters) (*ent.File, error) {
	stm := f.client.File.
		Create().
		SetOwnerID(args.Owner).
		SetType(int(types.FileTypeFolder)).
		SetIsSymbolic(args.IsSymbolic).
		SetName(args.Name)
	if root != nil {
		stm.SetParent(root).SetType(int(types.FileTypeFolder))
	}

	fid, err := stm.OnConflict(sql.ConflictColumns(file.FieldFileChildren, file.FieldName)).Ignore().ID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	newFolder, err := f.client.File.Get(ctx, fid)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	if len(args.Metadata) > 0 {
		_, err := f.client.Metadata.
			CreateBulk(lo.MapToSlice(args.Metadata, func(key, value string) *ent.MetadataCreate {
				_, isPrivate := args.MetadataPrivateMask[key]
				return f.client.Metadata.Create().
					SetName(key).
					SetValue(value).
					SetFile(newFolder).
					SetIsPublic(!isPrivate)
			})...).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create metadata: %v", err)
		}
	}

	return newFolder, err

}

func (f *fileClient) Rename(ctx context.Context, original *ent.File, newName string) (*ent.File, error) {
	return f.client.File.UpdateOne(original).SetName(newName).Save(ctx)
}

func (f *fileClient) GetEntitiesByIDs(ctx context.Context, ids []int, page int) ([]*ent.Entity, int, error) {
	groups, _ := f.batchInConditionEntityID(intsets.MaxInt, 10, 1, ids)
	if page >= len(groups) || page < 0 {
		return nil, -1, fmt.Errorf("page out of range")
	}

	res, err := f.client.Entity.Query().Where(groups[page]).All(ctx)
	if err != nil {
		return nil, page, err
	}

	if page+1 >= len(groups) {
		return res, -1, nil
	}

	return res, page + 1, nil
}

func (f *fileClient) GetEntityByID(ctx context.Context, id int) (*ent.Entity, error) {
	return withEntityEagerLoading(ctx, f.client.Entity.Query().Where(entity.ID(id))).First(ctx)
}

func (f *fileClient) GetByID(ctx context.Context, ids int) (*ent.File, error) {
	return withFileEagerLoading(ctx, f.client.File.Query().Where(file.ID(ids))).First(ctx)
}

func (f *fileClient) GetByIDs(ctx context.Context, ids []int, page int) ([]*ent.File, int, error) {
	groups, _ := f.batchInCondition(intsets.MaxInt, 10, 1, ids)
	if page >= len(groups) || page < 0 {
		return nil, -1, fmt.Errorf("page out of range")
	}

	res, err := withFileEagerLoading(ctx, f.client.File.Query().Where(groups[page])).All(ctx)
	if err != nil {
		return nil, page, err
	}

	if page+1 >= len(groups) {
		return res, -1, nil
	}

	return res, page + 1, nil
}

func (f *fileClient) FlattenListFiles(ctx context.Context, args *FlattenListFileParameters) (*ListFileResult, error) {
	query := f.client.File.Query().Where(file.Type(int(types.FileTypeFile)), file.IsSymbolic(false))

	if args.UserID > 0 {
		query = query.Where(file.OwnerID(args.UserID))
	}

	if args.StoragePolicyID > 0 {
		query = query.Where(file.StoragePolicyFiles(args.StoragePolicyID))
	}

	if args.Name != "" {
		query = query.Where(file.NameContainsFold(args.Name))
	}

	query.Order(getFileOrderOption(&ListFileParameters{
		PaginationArgs: args.PaginationArgs,
	})...)

	// Count total items
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count files: %w", err)
	}

	files, err := withFileEagerLoading(ctx, query).Limit(args.PageSize).Offset(args.Page * args.PageSize).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return &ListFileResult{
		Files: files,
		PaginationResults: &PaginationResults{
			TotalItems: total,
			Page:       args.Page,
			PageSize:   args.PageSize,
		},
	}, nil
}

func (f *fileClient) ListEntities(ctx context.Context, args *ListEntityParameters) (*ListEntityResult, error) {
	query := f.client.Entity.Query()
	if args.EntityType != nil {
		query = query.Where(entity.Type(int(*args.EntityType)))
	}

	if args.UserID > 0 {
		query = query.Where(entity.CreatedBy(args.UserID))
	}

	if args.StoragePolicyID > 0 {
		query = query.Where(entity.StoragePolicyEntities(args.StoragePolicyID))
	}

	query.Order(getEntityOrderOption(args)...)

	// Count total items
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count files: %w", err)
	}

	entities, err := withEntityEagerLoading(ctx, query).Limit(args.PageSize).Offset(args.Page * args.PageSize).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}

	return &ListEntityResult{
		Entities: entities,
		PaginationResults: &PaginationResults{
			TotalItems: total,
			Page:       args.Page,
			PageSize:   args.PageSize,
		},
	}, nil
}
