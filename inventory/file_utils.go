package inventory

import (
	"context"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/entity"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/ent/metadata"
	"github.com/cloudreve/Cloudreve/v4/ent/predicate"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

const (
	metadataExactMatchPrefix = "!exact:"
)

func (f *fileClient) searchQuery(q *ent.FileQuery, args *SearchFileParameters, parents []*ent.File, ownerId int) *ent.FileQuery {
	if len(parents) == 1 && parents[0] == nil {
		q = q.Where(file.OwnerID(ownerId))
	} else {
		q = q.Where(
			file.HasParentWith(
				file.IDIn(lo.Map(parents, func(item *ent.File, index int) int {
					return item.ID
				})...,
				),
			),
		)
	}

	if len(args.Name) > 0 {
		namePredicates := lo.Map(args.Name, func(item string, index int) predicate.File {
			// If start and ends with quotes, treat as exact match
			if strings.HasPrefix(item, "\"") && strings.HasSuffix(item, "\"") {
				return file.NameContains(strings.Trim(item, "\""))
			}

			// if contain wildcard, use transform to sql like
			if strings.Contains(item, SearchWildcard) {
				pattern := strings.ReplaceAll(item, SearchWildcard, "%")
				if pattern[0] != '%' && pattern[len(pattern)-1] != '%' {
					// if not start with wildcard, add prefix wildcard
					pattern = "%" + pattern + "%"
				}

				return func(s *sql.Selector) {
					s.Where(sql.Like(file.FieldName, pattern))
				}
			}

			if args.CaseFolding {
				return file.NameContainsFold(item)
			}

			return file.NameContains(item)
		})

		if args.NameOperatorOr {
			q = q.Where(file.Or(namePredicates...))
		} else {
			q = q.Where(file.And(namePredicates...))
		}
	}

	if args.Type != nil {
		q = q.Where(file.TypeEQ(int(*args.Type)))
	}

	if len(args.Metadata) > 0 {
		metaPredicates := lo.Map(args.Metadata, func(item MetadataFilter, index int) predicate.Metadata {
			if item.Exact {
				return metadata.And(metadata.NameEQ(item.Key), metadata.ValueEQ(item.Value))
			}

			nameEq := metadata.And(metadata.IsPublic(true), metadata.NameEQ(item.Key))
			if item.Value == "" {
				return nameEq
			} else {
				valueContain := metadata.ValueContainsFold(item.Value)
				return metadata.And(nameEq, valueContain)
			}
		})
		q.Where(file.And(lo.Map(metaPredicates, func(item predicate.Metadata, index int) predicate.File {
			return file.HasMetadataWith(item)
		})...))
	}

	if args.SizeLte > 0 || args.SizeGte > 0 {
		q = q.Where(file.SizeGTE(args.SizeGte), file.SizeLTE(args.SizeLte))
	}

	if args.CreatedAtLte != nil {
		q = q.Where(file.CreatedAtLTE(*args.CreatedAtLte))
	}

	if args.CreatedAtGte != nil {
		q = q.Where(file.CreatedAtGTE(*args.CreatedAtGte))
	}

	if args.UpdatedAtLte != nil {
		q = q.Where(file.UpdatedAtLTE(*args.UpdatedAtLte))
	}

	if args.UpdatedAtGte != nil {
		q = q.Where(file.UpdatedAtGTE(*args.UpdatedAtGte))
	}

	return q
}

// ChildFileQuery generates query for child file(s) of a given set of root
func (f *fileClient) childFileQuery(ownerID int, isSymbolic bool, root ...*ent.File) *ent.FileQuery {
	rawQuery := f.client.File.Query()
	if len(root) == 1 && root[0] != nil {
		// Query children of one single root
		rawQuery = f.client.File.QueryChildren(root[0])
	} else if root[0] == nil {
		// Query orphan files with owner ID
		predicates := []predicate.File{
			file.NameNEQ(RootFolderName),
		}

		if ownerID > 0 {
			predicates = append(predicates, file.OwnerIDEQ(ownerID))
		}

		if isSymbolic {
			predicates = append(predicates, file.And(file.IsSymbolic(true), file.FileChildrenNotNil()))
		} else {
			predicates = append(predicates, file.Not(file.HasParent()))
		}

		rawQuery = f.client.File.Query().Where(
			file.And(predicates...),
		)
	} else {
		// Query children of multiple roots
		rawQuery.
			Where(
				file.HasParentWith(
					file.IDIn(lo.Map(root, func(item *ent.File, index int) int {
						return item.ID
					})...),
				),
			)
	}

	return rawQuery
}

// batchInCondition returns a list of predicates that divide original group into smaller ones
// to bypass DB limitations.
func (f *fileClient) batchInCondition(pageSize, margin int, multiply int, ids []int) ([]predicate.File, [][]int) {
	pageSize = capPageSize(f.maxSQlParam, pageSize, margin)
	chunks := lo.Chunk(ids, max(pageSize/multiply, 1))
	return lo.Map(chunks, func(item []int, index int) predicate.File {
		return file.IDIn(item...)
	}), chunks
}

func (f *fileClient) batchInConditionMetadataName(pageSize, margin int, multiply int, keys []string) ([]predicate.Metadata, [][]string) {
	pageSize = capPageSize(f.maxSQlParam, pageSize, margin)
	chunks := lo.Chunk(keys, max(pageSize/multiply, 1))
	return lo.Map(chunks, func(item []string, index int) predicate.Metadata {
		return metadata.NameIn(item...)
	}), chunks
}

func (f *fileClient) batchInConditionEntityID(pageSize, margin int, multiply int, keys []int) ([]predicate.Entity, [][]int) {
	pageSize = capPageSize(f.maxSQlParam, pageSize, margin)
	chunks := lo.Chunk(keys, max(pageSize/multiply, 1))
	return lo.Map(chunks, func(item []int, index int) predicate.Entity {
		return entity.IDIn(item...)
	}), chunks
}

// cursorPagination perform pagination with cursor, which is faster than fast pagination, but less flexible.
func (f *fileClient) cursorPagination(ctx context.Context, query *ent.FileQuery,
	args *ListFileParameters, paramMargin int) ([]*ent.File, *PaginationResults, error) {
	pageSize := capPageSize(f.maxSQlParam, args.PageSize, paramMargin)
	query.Order(getFileOrderOption(args)...)
	currentPage := 0
	// Three types of query option
	queryPaged := []*ent.FileQuery{
		query.Clone().
			Where(file.TypeEQ(int(types.FileTypeFolder))),
		query.Clone().
			Where(file.TypeEQ(int(types.FileTypeFile))),
		query.Clone().
			Where(file.TypeIn(int(types.FileTypeFolder), int(types.FileTypeFile))),
	}

	var (
		pageToken *PageToken
		err       error
	)
	if args.PageToken != "" {
		pageToken, err = pageTokenFromString(args.PageToken, f.hasher, hashid.FileID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page token %q: %w", args.PageToken, err)
		}
	}
	queryPaged = getFileCursorQuery(args, pageToken, queryPaged)

	// Use page size + 1 to determine if there are more items to come
	queryPaged[0].Limit(pageSize + 1)

	files, err := queryPaged[0].
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	nextStartWithFile := false
	if pageToken != nil && pageToken.StartWithFile {
		nextStartWithFile = true
	}
	if len(files) < pageSize+1 && len(queryPaged) > 1 && !args.MixedType && !args.FolderOnly {
		queryPaged[1].Limit(pageSize + 1 - len(files))
		filesContinue, err := queryPaged[1].
			All(ctx)
		if err != nil {
			return nil, nil, err
		}

		nextStartWithFile = true
		files = append(files, filesContinue...)
	}

	// More items to come
	nextTokenStr := ""
	if len(files) > pageSize {
		lastItem := files[len(files)-2]
		nextToken, err := getFileNextPageToken(f.hasher, lastItem, args, nextStartWithFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate next page token: %w", err)
		}

		nextTokenStr = nextToken
	}

	return lo.Subset(files, 0, uint(pageSize)), &PaginationResults{
		Page:          currentPage,
		PageSize:      pageSize,
		NextPageToken: nextTokenStr,
		IsCursor:      true,
	}, nil

}

// offsetPagination perform traditional pagination with minor optimizations.
func (f *fileClient) offsetPagination(ctx context.Context, query *ent.FileQuery,
	args *ListFileParameters, paramMargin int) ([]*ent.File, *PaginationResults, error) {
	pageSize := capPageSize(f.maxSQlParam, args.PageSize, paramMargin)
	queryWithoutOrder := query.Clone()
	query.Order(getFileOrderOption(args)...)

	// Count total items by type
	var v []struct {
		Type  int `json:"type"`
		Count int `json:"count"`
	}
	err := queryWithoutOrder.Clone().
		GroupBy(file.FieldType).
		Aggregate(ent.Count()).
		Scan(ctx, &v)
	if err != nil {
		return nil, nil, err
	}

	folderCount := 0
	fileCount := 0
	for _, item := range v {
		if item.Type == int(types.FileTypeFolder) {
			folderCount = item.Count
		} else {
			fileCount = item.Count
		}
	}

	allFiles := make([]*ent.File, 0, pageSize)
	folderLimit := 0
	if (args.Page+1)*pageSize > folderCount {
		folderLimit = folderCount - args.Page*pageSize
		if folderLimit < 0 {
			folderLimit = 0
		}
	} else {
		folderLimit = pageSize
	}

	if folderLimit <= pageSize && folderLimit > 0 {
		// Folder still remains
		folders, err := query.Clone().
			Limit(folderLimit).
			Offset(args.Page * pageSize).
			Where(file.TypeEQ(int(types.FileTypeFolder))).All(ctx)
		if err != nil {
			return nil, nil, err
		}

		allFiles = append(allFiles, folders...)
	}

	if folderLimit < pageSize {
		files, err := query.Clone().
			Limit(pageSize - folderLimit).
			Offset((args.Page * pageSize) + folderLimit - folderCount).
			Where(file.TypeEQ(int(types.FileTypeFile))).
			All(ctx)
		if err != nil {
			return nil, nil, err
		}

		allFiles = append(allFiles, files...)
	}

	return allFiles, &PaginationResults{
		TotalItems: folderCount + fileCount,
		Page:       args.Page,
		PageSize:   pageSize,
	}, nil
}

func withFileEagerLoading(ctx context.Context, q *ent.FileQuery) *ent.FileQuery {
	if v, ok := ctx.Value(LoadFileEntity{}).(bool); ok && v {
		q.WithEntities(func(m *ent.EntityQuery) {
			m.Order(ent.Desc(entity.FieldID))
			withEntityEagerLoading(ctx, m)
		})
	}
	if v, ok := ctx.Value(LoadFileMetadata{}).(bool); ok && v {
		q.WithMetadata()
	}
	if v, ok := ctx.Value(LoadFilePublicMetadata{}).(bool); ok && v {
		q.WithMetadata(func(m *ent.MetadataQuery) {
			m.Where(metadata.IsPublic(true))
		})
	}
	if v, ok := ctx.Value(LoadFileShare{}).(bool); ok && v {
		q.WithShares()
	}
	if v, ok := ctx.Value(LoadFileUser{}).(bool); ok && v {
		q.WithOwner(func(query *ent.UserQuery) {
			withUserEagerLoading(ctx, query)
		})
	}
	if v, ok := ctx.Value(LoadFileDirectLink{}).(bool); ok && v {
		q.WithDirectLinks()
	}

	return q
}

func withEntityEagerLoading(ctx context.Context, q *ent.EntityQuery) *ent.EntityQuery {
	if v, ok := ctx.Value(LoadEntityUser{}).(bool); ok && v {
		q.WithUser()
	}

	if v, ok := ctx.Value(LoadEntityStoragePolicy{}).(bool); ok && v {
		q.WithStoragePolicy()
	}

	if v, ok := ctx.Value(LoadEntityFile{}).(bool); ok && v {
		q.WithFile(func(fq *ent.FileQuery) {
			withFileEagerLoading(ctx, fq)
		})
	}

	return q
}

func getFileOrderOption(args *ListFileParameters) []file.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case file.FieldName:
		return []file.OrderOption{file.ByName(orderTerm), file.ByID(orderTerm)}
	case file.FieldSize:
		return []file.OrderOption{file.BySize(orderTerm), file.ByID(orderTerm)}
	case file.FieldUpdatedAt:
		return []file.OrderOption{file.ByUpdatedAt(orderTerm), file.ByID(orderTerm)}
	default:
		return []file.OrderOption{file.ByID(orderTerm)}
	}
}

func getEntityOrderOption(args *ListEntityParameters) []entity.OrderOption {
	orderTerm := getOrderTerm(args.Order)
	switch args.OrderBy {
	case entity.FieldSize:
		return []entity.OrderOption{entity.BySize(orderTerm), entity.ByID(orderTerm)}
	case entity.FieldUpdatedAt:
		return []entity.OrderOption{entity.ByUpdatedAt(orderTerm), entity.ByID(orderTerm)}
	case entity.FieldReferenceCount:
		return []entity.OrderOption{entity.ByReferenceCount(orderTerm), entity.ByID(orderTerm)}
	default:
		return []entity.OrderOption{entity.ByID(orderTerm)}
	}
}

var fileCursorQuery = map[string]map[bool]func(token *PageToken) predicate.File{
	file.FieldName: {
		true: func(token *PageToken) predicate.File {
			return file.Or(
				file.NameLT(token.String),
				file.And(file.Name(token.String), file.IDLT(token.ID)),
			)
		},
		false: func(token *PageToken) predicate.File {
			return file.Or(
				file.NameGT(token.String),
				file.And(file.Name(token.String), file.IDGT(token.ID)),
			)
		},
	},
	file.FieldSize: {
		true: func(token *PageToken) predicate.File {
			return file.Or(
				file.SizeLT(int64(token.Int)),
				file.And(file.Size(int64(token.Int)), file.IDLT(token.ID)),
			)
		},
		false: func(token *PageToken) predicate.File {
			return file.Or(
				file.SizeGT(int64(token.Int)),
				file.And(file.Size(int64(token.Int)), file.IDGT(token.ID)),
			)
		},
	},
	file.FieldCreatedAt: {
		true: func(token *PageToken) predicate.File {
			return file.IDLT(token.ID)
		},
		false: func(token *PageToken) predicate.File {
			return file.IDGT(token.ID)
		},
	},
	file.FieldUpdatedAt: {
		true: func(token *PageToken) predicate.File {
			return file.Or(
				file.UpdatedAtLT(*token.Time),
				file.And(file.UpdatedAt(*token.Time), file.IDLT(token.ID)),
			)
		},
		false: func(token *PageToken) predicate.File {
			return file.Or(
				file.UpdatedAtGT(*token.Time),
				file.And(file.UpdatedAt(*token.Time), file.IDGT(token.ID)),
			)
		},
	},
	file.FieldID: {
		true: func(token *PageToken) predicate.File {
			return file.IDLT(token.ID)
		},
		false: func(token *PageToken) predicate.File {
			return file.IDGT(token.ID)
		},
	},
}

func getFileCursorQuery(args *ListFileParameters, token *PageToken, query []*ent.FileQuery) []*ent.FileQuery {
	o := &sql.OrderTermOptions{}
	getOrderTerm(args.Order)(o)

	predicates, ok := fileCursorQuery[args.OrderBy]
	if !ok {
		predicates = fileCursorQuery[file.FieldID]
	}

	// If all folder is already listed in previous page, only query for files.
	if token != nil && token.StartWithFile && !args.MixedType {
		query = query[1:2]
	}

	// Mixing folders and files with one query
	if args.MixedType {
		query = query[2:]
	} else if args.FolderOnly {
		query = query[0:1]
	}

	if token != nil {
		query[0].Where(predicates[o.Desc](token))
	}
	return query
}

// getFileNextPageToken returns the next page token for the given last file.
func getFileNextPageToken(hasher hashid.Encoder, last *ent.File, args *ListFileParameters, nextStartWithFile bool) (string, error) {
	token := &PageToken{
		ID:            last.ID,
		StartWithFile: nextStartWithFile,
	}

	switch args.OrderBy {
	case file.FieldName:
		token.String = last.Name
	case file.FieldSize:
		token.Int = int(last.Size)
	case file.FieldUpdatedAt:
		token.Time = &last.UpdatedAt
	}

	return token.Encode(hasher, hashid.EncodeFileID)
}
