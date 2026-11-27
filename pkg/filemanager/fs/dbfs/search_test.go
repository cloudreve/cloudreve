package dbfs

import (
	"context"
	"sort"
	"strconv"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type searchPaginationClient struct {
	inventory.FileClient
	folders []*ent.File
	files   []*ent.File
}

func (c *searchPaginationClient) GetChildFiles(_ context.Context, args *inventory.ListFileParameters,
	_ int, roots ...*ent.File) (*inventory.ListFileResult, error) {
	parents := make(map[int]bool, len(roots))
	for _, root := range roots {
		parents[root.ID] = true
	}
	var models []*ent.File
	if args.Search != nil {
		models = c.files
	} else {
		models = c.folders
		if !args.FolderOnly {
			models = append(append([]*ent.File(nil), models...), c.files...)
		}
	}
	cursor, _ := strconv.Atoi(args.PageToken)
	matches := make([]*ent.File, 0)
	for _, item := range models {
		if parents[item.FileChildren] && item.ID > cursor {
			matches = append(matches, item)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ID < matches[j].ID })
	size := args.PageSize
	if size == 0 || size > 2 {
		size = 2
	}
	result := &inventory.ListFileResult{Files: matches, PaginationResults: &inventory.PaginationResults{}}
	if len(matches) > size {
		result.Files = matches[:size]
		result.NextPageToken = strconv.Itoa(result.Files[size-1].ID)
	}
	return result, nil
}

func newSearchPaginationFixture(t *testing.T, limit int) (*baseNavigator, *File) {
	t.Helper()
	client := &searchPaginationClient{
		folders: []*ent.File{
			{ID: 2, Name: "a", FileChildren: 1, OwnerID: 42, Type: int(types.FileTypeFolder)},
			{ID: 3, Name: "b", FileChildren: 1, OwnerID: 42, Type: int(types.FileTypeFolder)},
			{ID: 4, Name: "c", FileChildren: 1, OwnerID: 42, Type: int(types.FileTypeFolder)},
		},
		files: []*ent.File{
			{ID: 5, Name: "first.txt", FileChildren: 2, OwnerID: 42, Type: int(types.FileTypeFile)},
			{ID: 6, Name: "last.txt", FileChildren: 4, OwnerID: 42, Type: int(types.FileTypeFile)},
		},
	}
	owner := &ent.User{ID: 42}
	root := newFile(nil, &ent.File{ID: 1, Name: "Photos", OwnerID: owner.ID, Type: int(types.FileTypeFolder)})
	root.OwnerModel = owner
	uri, err := fs.NewUriFromString("cloudreve://my/Photos")
	require.NoError(t, err)
	root.Path = [2]*fs.URI{uri, uri}
	t.Cleanup(root.Recycle)
	return newBaseNavigator(client, defaultFilter, owner, nil,
		&setting.DBFS{MaxRecursiveSearchedFolder: limit}), root
}

func TestSearchMergesDirectoryPagesWithinBudget(t *testing.T) {
	for _, test := range []struct {
		limit     int
		want      []int
		truncated bool
	}{
		{limit: 4, want: []int{5, 6}},
		{limit: 3, want: []int{5}, truncated: true},
		{limit: 2, want: []int{5}, truncated: true},
	} {
		t.Run(strconv.Itoa(test.limit), func(t *testing.T) {
			navigator, root := newSearchPaginationFixture(t, test.limit)
			result, err := navigator.search(context.Background(), root, &ListArgs{
				Page: &inventory.PaginationArgs{PageSize: 10}, Search: &inventory.SearchFileParameters{Name: []string{"txt"}},
			})
			require.NoError(t, err)
			var ids []int
			for _, item := range result.Files {
				ids = append(ids, item.ID())
			}
			require.Equal(t, test.want, ids)
			require.Equal(t, test.truncated, result.RecursionLimitReached)
			require.Empty(t, result.Pagination.NextPageToken)
		})
	}
}

func TestSearchPaginationPreservesFiltering(t *testing.T) {
	navigator, root := newSearchPaginationFixture(t, 4)
	navigator.listFilter = func(_ context.Context, file *File) (*File, bool) {
		return file, file.ID() != 5
	}
	var ids []int
	token := ""
	for page := 0; ; page++ {
		require.Less(t, page, 5)
		result, err := navigator.search(context.Background(), root, &ListArgs{
			Page:   &inventory.PaginationArgs{PageSize: 1, PageToken: token},
			Search: &inventory.SearchFileParameters{Name: []string{"txt"}},
		})
		require.NoError(t, err)
		require.False(t, result.RecursionLimitReached)
		for _, item := range result.Files {
			ids = append(ids, item.ID())
		}
		token = result.Pagination.NextPageToken
		if token == "" {
			break
		}
	}
	require.Equal(t, []int{6}, ids)
}

func TestSearchPaginationRetainsCollectedFolders(t *testing.T) {
	for _, test := range []struct {
		limit     int
		want      []int
		truncated bool
	}{
		{limit: 5, want: []int{5, 6, 7, 9}},
		{limit: 3, want: []int{5, 7}, truncated: true},
	} {
		t.Run(strconv.Itoa(test.limit), func(t *testing.T) {
			navigator, root := newSearchPaginationFixture(t, test.limit)
			client := navigator.fileClient.(*searchPaginationClient)
			client.folders = append(client.folders,
				&ent.File{ID: 8, Name: "deep", FileChildren: 4, OwnerID: 42, Type: int(types.FileTypeFolder)})
			client.files = append(client.files,
				&ent.File{ID: 7, Name: "another.txt", FileChildren: 2, OwnerID: 42, Type: int(types.FileTypeFile)},
				&ent.File{ID: 9, Name: "deep.txt", FileChildren: 8, OwnerID: 42, Type: int(types.FileTypeFile)})
			var ids []int
			token := ""
			for page := 0; ; page++ {
				require.Less(t, page, 10)
				result, err := navigator.search(context.Background(), root, &ListArgs{
					Page:   &inventory.PaginationArgs{PageSize: 1, PageToken: token},
					Search: &inventory.SearchFileParameters{Name: []string{"txt"}},
				})
				require.NoError(t, err)
				require.Equal(t, test.truncated, result.RecursionLimitReached)
				for _, item := range result.Files {
					ids = append(ids, item.ID())
				}
				token = result.Pagination.NextPageToken
				if token == "" {
					break
				}
			}
			require.Equal(t, test.want, ids)
		})
	}
}

func TestWalkAdvancesDirectoryCursor(t *testing.T) {
	navigator, root := newSearchPaginationFixture(t, 20)
	var ids []int
	err := navigator.walk(context.Background(), []*File{root}, 20, 10, func(files []*File, _ int) error {
		for _, file := range files {
			ids = append(ids, file.ID())
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3, 4, 5, 6}, ids)
}
