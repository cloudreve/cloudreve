package inventory

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/file"
	_ "github.com/cloudreve/Cloudreve/v4/ent/runtime"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/stretchr/testify/require"
)

type fileSearchDriver struct {
	dialect.Driver
	kind    string
	mu      sync.Mutex
	limit   int
	args    [][]any
	failAt  int
	failErr error
}

func (d *fileSearchDriver) Dialect() string { return d.kind }
func (d *fileSearchDriver) Query(ctx context.Context, query string, args, rows any) error {
	parameters := args.([]any)
	d.mu.Lock()
	d.args = append(d.args, append([]any(nil), parameters...))
	tooMany := d.limit > 0 && len(parameters) > d.limit
	fail := d.failAt > 0 && len(d.args) == d.failAt
	d.mu.Unlock()
	if fail {
		return d.failErr
	}
	if tooMany {
		return fmt.Errorf("query exceeds the test bind limit: %d", len(parameters))
	}
	if d.Driver == nil {
		*rows.(*entsql.Rows) = entsql.Rows{ColumnScanner: emptyFileSearchRows{}}
		return nil
	}
	return d.Driver.Query(ctx, query, args, rows)
}

type emptyFileSearchRows struct{}

func (emptyFileSearchRows) Close() error                               { return nil }
func (emptyFileSearchRows) ColumnTypes() ([]*stdsql.ColumnType, error) { return nil, nil }
func (emptyFileSearchRows) Columns() ([]string, error)                 { return file.Columns, nil }
func (emptyFileSearchRows) Err() error                                 { return nil }
func (emptyFileSearchRows) Next() bool                                 { return false }
func (emptyFileSearchRows) NextResultSet() bool                        { return false }
func (emptyFileSearchRows) Scan(...any) error                          { return nil }

func TestGetChildFilesBoundsParentParameters(t *testing.T) {
	for _, db := range []struct {
		kind string
		conf conf.DBType
	}{
		{dialect.Postgres, conf.PostgresDB},
		{dialect.SQLite, conf.SQLiteDB},
		{dialect.MySQL, conf.MySqlDB},
	} {
		for _, count := range []int{2, 32767, 65534} {
			t.Run(fmt.Sprintf("%s/%d", db.kind, count), func(t *testing.T) {
				driver := &fileSearchDriver{kind: db.kind, limit: sqlParamLimit(db.conf)}
				client := NewFileClient(ent.NewClient(ent.Driver(driver)), db.conf, nil)
				roots := make([]*ent.File, count)
				for i := range roots {
					roots[i] = &ent.File{ID: 100000 + i}
				}
				result, err := client.GetChildFiles(context.Background(), &ListFileParameters{
					PaginationArgs: &PaginationArgs{PageSize: 2},
					Search:         &SearchFileParameters{Name: []string{"needle"}},
					MixedType:      true,
				}, 1, roots...)
				require.NoError(t, err)
				require.Empty(t, result.Files)
				require.Len(t, driver.args, (count+29999)/30000)
				seen := make(map[int]int, count)
				for _, args := range driver.args {
					require.LessOrEqual(t, len(args), driver.limit)
					for _, arg := range args {
						if id, ok := arg.(int); ok && id >= 100000 {
							seen[id]++
						}
					}
				}
				require.Len(t, seen, count)
				for _, occurrences := range seen {
					require.Equal(t, 1, occurrences, "each parent should be bound once")
				}
			})
		}
	}
}

func TestGetChildFilesReservesSearchAndCursorParameters(t *testing.T) {
	driver := &fileSearchDriver{kind: dialect.Postgres, limit: 32}
	encoder, err := hashid.New("file-search-test")
	require.NoError(t, err)
	client := NewFileClient(ent.NewClient(ent.Driver(driver)), conf.PostgresDB, encoder).(*fileClient)
	client.maxSQlParam = driver.limit
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	kind := types.FileTypeFile
	search := &SearchFileParameters{
		Name: []string{"needle"}, Type: &kind, Metadata: []MetadataFilter{{Key: "tag", Value: "yes"}},
		SizeGte: 1, SizeLte: 10, CreatedAtGte: &date, CreatedAtLte: &date, UpdatedAtGte: &date, UpdatedAtLte: &date,
	}
	token, err := (&PageToken{ID: 1, Time: &date}).Encode(encoder, hashid.EncodeFileID)
	require.NoError(t, err)
	roots := make([]*ent.File, 24)
	for i := range roots {
		roots[i] = &ent.File{ID: i + 1}
	}
	_, err = client.GetChildFiles(context.Background(), &ListFileParameters{
		PaginationArgs: &PaginationArgs{PageSize: 2, OrderBy: file.FieldUpdatedAt, PageToken: token},
		MixedType:      true, Search: search,
	}, 1, roots...)
	require.NoError(t, err)
	require.Greater(t, len(driver.args), 1)
	for _, args := range driver.args {
		require.LessOrEqual(t, len(args), driver.limit)
	}
	search.Name = make([]string, driver.limit)
	result, err := client.GetChildFiles(context.Background(), &ListFileParameters{
		PaginationArgs: &PaginationArgs{PageSize: 2}, Search: search,
	}, 1, roots...)
	require.ErrorIs(t, err, ErrTooManyArguments)
	require.Nil(t, result)
}

type fileSearchFixture struct {
	client *ent.Client
	files  *fileClient
	driver *fileSearchDriver
	owner  int
	roots  []*ent.File
}

func newFileSearchFixture(t *testing.T) *fileSearchFixture {
	t.Helper()
	db, err := stdsql.Open("sqlite3", filepath.Join(t.TempDir(), "search.db"))
	require.NoError(t, err)
	driver := &fileSearchDriver{Driver: entsql.OpenDB(dialect.SQLite, db), kind: dialect.SQLite}
	client := ent.NewClient(ent.Driver(driver))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	require.NoError(t, client.Schema.Create(ctx))
	group, err := client.Group.Create().SetName("test").SetPermissions(&boolset.BooleanSet{}).Save(ctx)
	require.NoError(t, err)
	owner, err := client.User.Create().SetEmail("owner@example.com").SetNick("owner").SetGroupID(group.ID).Save(ctx)
	require.NoError(t, err)
	other, err := client.User.Create().SetEmail("other@example.com").SetNick("other").SetGroupID(group.ID).Save(ctx)
	require.NoError(t, err)
	for _, user := range []*ent.User{owner, other} {
		_, err := client.File.Create().SetName("needle-orphan").SetOwnerID(user.ID).SetType(int(types.FileTypeFile)).Save(ctx)
		require.NoError(t, err)
	}
	root, err := client.File.Create().SetName(RootFolderName).SetOwnerID(owner.ID).SetType(int(types.FileTypeFolder)).Save(ctx)
	require.NoError(t, err)
	roots := make([]*ent.File, 48)
	for i := range roots {
		roots[i], err = client.File.Create().SetName(fmt.Sprintf("parent-%02d", i)).
			SetOwnerID(owner.ID).SetParent(root).SetType(int(types.FileTypeFolder)).Save(ctx)
		require.NoError(t, err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := len(roots) - 1; i >= 0; i-- {
		for j := 0; j < 2; j++ {
			item, err := client.File.Create().
				SetName(fmt.Sprintf("needle-%s-%02d-%d", []string{"z", "A", "\u00e9"}[i%3], i, j)).
				SetOwnerID(owner.ID).SetParent(roots[i]).SetType(int(types.FileTypeFile)).
				SetSize(int64((i + j) % 5)).SetCreatedAt(base.Add(time.Duration((i+j)%4) * time.Second)).
				SetUpdatedAt(base.Add(time.Duration((i+j)%3) * time.Second)).Save(ctx)
			require.NoError(t, err)
			if j == 0 {
				_, err = client.Metadata.Create().SetFile(item).SetName("tag").SetValue("yes").SetIsPublic(true).Save(ctx)
				require.NoError(t, err)
			}
		}
		if i%3 == 0 {
			_, err := client.File.Create().SetName(fmt.Sprintf("needle-folder-%02d", i)).
				SetOwnerID(owner.ID).SetParent(roots[i]).SetType(int(types.FileTypeFolder)).Save(ctx)
			require.NoError(t, err)
		}
	}
	_, err = client.File.Create().SetName("needle-outside").SetOwnerID(owner.ID).SetParent(root).
		SetType(int(types.FileTypeFile)).Save(ctx)
	require.NoError(t, err)
	encoder, err := hashid.New("file-search-test")
	require.NoError(t, err)
	return &fileSearchFixture{client: client, files: NewFileClient(client, conf.SQLiteDB, encoder).(*fileClient),
		driver: driver, owner: owner.ID, roots: roots}
}

func TestGetChildFilesBatchedCursorOrdering(t *testing.T) {
	f := newFileSearchFixture(t)
	ctx := context.WithValue(context.Background(), LoadFilePublicMetadata{}, true)
	ids := make([]int, len(f.roots))
	for i, root := range f.roots {
		ids[i] = root.ID
	}
	for _, orderBy := range []string{file.FieldID, file.FieldName, file.FieldSize, file.FieldCreatedAt, file.FieldUpdatedAt} {
		for _, direction := range []OrderDirection{OrderDirectionAsc, OrderDirectionDesc} {
			t.Run(orderBy+"/"+string(direction), func(t *testing.T) {
				f.driver.limit = 0
				search := &SearchFileParameters{Name: []string{"needle"},
					Metadata: []MetadataFilter{{Key: "tag", Value: "yes", Exact: true}}}
				args := &ListFileParameters{PaginationArgs: &PaginationArgs{PageSize: 2, OrderBy: orderBy, Order: direction},
					MixedType: true, Search: search}
				query := f.client.File.Query().Where(file.FileChildrenIn(ids...))
				expected, err := f.files.searchQuery(query, search, f.roots, f.owner).
					Order(getFileOrderOption(args)...).Unique(false).IDs(ctx)
				require.NoError(t, err)
				f.driver.limit, f.files.maxSQlParam = 32, 32
				var got []int
				tokens := map[string]bool{}
				for {
					result, err := f.files.GetChildFiles(ctx, args, f.owner, append(f.roots, f.roots[0])...)
					require.NoError(t, err)
					require.LessOrEqual(t, len(result.Files), 2)
					for _, item := range result.Files {
						got = append(got, item.ID)
						require.Len(t, item.Edges.Metadata, 1)
					}
					if result.NextPageToken == "" {
						break
					}
					require.False(t, tokens[result.NextPageToken], "cursor must advance")
					tokens[result.NextPageToken] = true
					args.PageToken = result.NextPageToken
				}
				require.Equal(t, expected, got)
			})
		}
	}
}

func TestGetChildFilesBatchedFoldersAndSpecialRoots(t *testing.T) {
	f := newFileSearchFixture(t)
	ctx := context.Background()
	ids := make([]int, len(f.roots))
	for i, root := range f.roots {
		ids[i] = root.ID
	}
	for _, test := range []struct {
		name       string
		folderOnly bool
		mixed      bool
		pageSize   int
	}{
		{name: "folders first", pageSize: 8},
		{name: "folders only", folderOnly: true, pageSize: 8},
		{name: "bounded large page", mixed: true, pageSize: 1000},
	} {
		t.Run(test.name, func(t *testing.T) {
			f.driver.limit = 0
			query := f.client.File.Query().Where(file.FileChildrenIn(ids...))
			if test.folderOnly {
				query.Where(file.TypeEQ(int(types.FileTypeFolder)))
			}
			if !test.mixed {
				query.Order(file.ByType(entsql.OrderDesc()))
			}
			expected, err := query.Order(file.ByID()).Unique(false).IDs(ctx)
			require.NoError(t, err)
			f.driver.limit, f.files.maxSQlParam = 32, 32
			args := &ListFileParameters{
				PaginationArgs: &PaginationArgs{UseCursorPagination: true, PageSize: test.pageSize},
				FolderOnly:     test.folderOnly, MixedType: test.mixed,
			}
			var got []int
			for page := 0; ; page++ {
				require.Less(t, page, 20)
				result, err := f.files.GetChildFiles(ctx, args, f.owner, f.roots...)
				require.NoError(t, err)
				for _, item := range result.Files {
					got = append(got, item.ID)
				}
				if result.NextPageToken == "" {
					break
				}
				args.PageToken = result.NextPageToken
			}
			require.Equal(t, expected, got)
		})
	}
	args := &ListFileParameters{PaginationArgs: &PaginationArgs{PageSize: 10},
		MixedType: true, Search: &SearchFileParameters{Name: []string{"orphan"}}}
	result, err := f.files.GetChildFiles(ctx, args, f.owner, nil)
	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	require.Equal(t, f.owner, result.Files[0].OwnerID)
	result, err = f.files.GetChildFiles(ctx, args, 0, nil)
	require.NoError(t, err)
	require.Empty(t, result.Files)
}

func TestGetChildFilesDoesNotHideBatchErrors(t *testing.T) {
	f := newFileSearchFixture(t)
	f.driver.limit, f.files.maxSQlParam = 32, 32
	f.driver.args = nil
	f.driver.failAt = 3 // The first two groups succeed, then their merge fails.
	f.driver.failErr = errors.New("database unavailable")
	result, err := f.files.GetChildFiles(context.Background(), &ListFileParameters{
		PaginationArgs: &PaginationArgs{UseCursorPagination: true, PageSize: 2}, MixedType: true,
	}, f.owner, f.roots...)
	require.ErrorIs(t, err, f.driver.failErr)
	require.Nil(t, result)
}
