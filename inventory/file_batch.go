package inventory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

const childFileParamMargin = 10

func (f *fileClient) childFileParentBatchSize(search *SearchFileParameters) (int, error) {
	// Reserve room for type filters, cursor predicates and search arguments.
	margin := childFileParamMargin
	if search != nil {
		margin += len(search.Name) + 3*len(search.Metadata)
		if search.Type != nil {
			margin++
		}
		if search.SizeGte > 0 || search.SizeLte > 0 {
			margin += 2
		}
		for _, date := range []*time.Time{search.CreatedAtGte, search.CreatedAtLte, search.UpdatedAtGte, search.UpdatedAtLte} {
			if date != nil {
				margin++
			}
		}
	}
	size := capPageSize(f.maxSQlParam, 30000, margin)
	if size < 1 {
		return 0, ErrTooManyArguments
	}
	return size, nil
}

func (f *fileClient) batchedChildFileCursor(ctx context.Context, args *ListFileParameters, ownerID int,
	roots []*ent.File, batchSize int) ([]*ent.File, *PaginationResults, error) {
	// Each merge combines at most two candidate pages in one bounded ID query.
	pageSize := min(capPageSize(f.maxSQlParam, args.PageSize, childFileParamMargin),
		(f.maxSQlParam-childFileParamMargin)/2-1)
	if pageSize < 1 {
		return nil, nil, ErrTooManyArguments
	}
	var token *PageToken
	if args.PageToken != "" {
		var err error
		token, err = pageTokenFromString(args.PageToken, f.hasher, hashid.FileID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page token %q: %w", args.PageToken, err)
		}
	}

	orderedRoots := append([]*ent.File(nil), roots...)
	sort.Slice(orderedRoots, func(i, j int) bool { return orderedRoots[i].ID < orderedRoots[j].ID })
	groups := lo.Chunk(orderedRoots, batchSize)
	queries := make([][]*ent.FileQuery, len(groups))
	for i, group := range groups {
		queries[i] = fileCursorQueries(f.scopedChildFileQuery(ctx, args, ownerID, group...), args, token)
	}

	files := make([]*ent.File, 0, pageSize+1)
	for phase := range queries[0] {
		left := pageSize + 1 - len(files)
		if left == 0 {
			break
		}
		var candidates []*ent.File
		for _, group := range queries {
			page, err := group[phase].Limit(left).All(ctx)
			if err != nil {
				return nil, nil, err
			}
			candidates, err = f.mergeChildFilePages(ctx, candidates, page, args, left)
			if err != nil {
				return nil, nil, err
			}
		}
		files = append(files, candidates...)
	}

	next := ""
	if len(files) > pageSize {
		last := files[pageSize-1]
		startWithFile := !args.MixedType && !args.FolderOnly && last.Type == int(types.FileTypeFile)
		var err error
		next, err = getFileNextPageToken(f.hasher, last, args, startWithFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate next page token: %w", err)
		}
		files = files[:pageSize]
	}
	return files, &PaginationResults{PageSize: pageSize, NextPageToken: next, IsCursor: true}, nil
}

func (f *fileClient) mergeChildFilePages(ctx context.Context, left, right []*ent.File,
	args *ListFileParameters, limit int) ([]*ent.File, error) {
	if len(left) == 0 {
		return right, nil
	}
	if len(right) == 0 {
		return left, nil
	}
	candidates := make(map[int]*ent.File, len(left)+len(right))
	for _, page := range [][]*ent.File{left, right} {
		for _, item := range page {
			candidates[item.ID] = item
		}
	}
	ids := lo.Keys(candidates)
	// Let the database order the bounded candidate set, including its collation.
	ordered, err := f.client.File.Query().
		Where(file.IDIn(ids...)).
		Order(getFileOrderOption(args)...).
		Unique(false).
		Limit(limit).
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*ent.File, 0, len(ordered))
	for _, id := range ordered {
		item, ok := candidates[id]
		if !ok {
			return nil, fmt.Errorf("unexpected file in merged cursor page")
		}
		result = append(result, item)
	}
	return result, nil
}
