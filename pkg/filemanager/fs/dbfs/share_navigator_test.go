package dbfs

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type shareNavigatorTestFileClient struct {
	inventory.FileClient
	shared       *ent.File
	children     map[int]map[string]*ent.File
	childLookups int
}

func (c *shareNavigatorTestFileClient) GetByID(_ context.Context, id int) (*ent.File, error) {
	if id == c.shared.ID {
		return c.shared, nil
	}
	return nil, fs.ErrPathNotExist
}

func (c *shareNavigatorTestFileClient) GetChildFile(_ context.Context, root *ent.File, _ int, name string,
	_ bool) (*ent.File, error) {
	c.childLookups++
	if child, ok := c.children[root.ID][name]; ok {
		return child, nil
	}
	return nil, fs.ErrPathNotExist
}

func newShareNavigatorForTest(t *testing.T, singleFile bool, userID int) (*shareNavigator, *shareNavigatorTestFileClient) {
	t.Helper()
	owner := &ent.User{ID: 1}
	user := &ent.User{ID: userID}
	root := newFile(nil, &ent.File{ID: 10, Name: "docs", Type: int(types.FileTypeFolder), OwnerID: owner.ID})
	t.Cleanup(root.Recycle)
	root.OwnerModel = owner
	root.IsUserRoot = true
	var err error
	root.Path[pathIndexUser], err = fs.NewUriFromString("cloudreve://test-share@share/")
	require.NoError(t, err)
	shared := &ent.File{ID: 11, Name: "shared.txt", Type: int(types.FileTypeFile), OwnerID: owner.ID}
	private := &ent.File{ID: 12, Name: "private", Type: int(types.FileTypeFolder), OwnerID: owner.ID}
	secret := &ent.File{ID: 13, Name: "secret.txt", Type: int(types.FileTypeFile), OwnerID: owner.ID}
	client := &shareNavigatorTestFileClient{
		shared: shared,
		children: map[int]map[string]*ent.File{
			root.ID():  {shared.Name: shared, private.Name: private},
			private.ID: {secret.Name: secret},
		},
	}
	n := NewShareNavigator(user, client, nil, nil, &setting.DBFS{}, nil).(*shareNavigator)
	n.shareRoot = root
	n.ownerRoot = root
	n.owner = owner
	n.singleFileShare = singleFile
	n.share = &ent.Share{Edges: ent.ShareEdges{File: shared}}
	if !singleFile {
		n.share.Edges.File = root.Model
	}
	return n, client
}

func TestShareNavigatorSingleFileBoundary(t *testing.T) {
	tests := []struct {
		name string
		path string
		id   int
	}{
		{name: "bare root", path: "", id: 10},
		{name: "root slash", path: "/", id: 10},
		{name: "shared file", path: "/shared.txt", id: 11},
		{name: "wrong filename", path: "/secret.txt"},
		{name: "nested sibling", path: "/private/secret.txt"},
		{name: "encoded nested sibling", path: "/private%2Fsecret.txt"},
		{name: "below shared file", path: "/shared.txt/secret.txt"},
	}
	for _, actor := range []struct {
		name string
		id   int
	}{
		{name: "anonymous", id: 0},
		{name: "recipient", id: 2},
		{name: "owner", id: 1},
	} {
		for _, restored := range []bool{false, true} {
			for _, test := range tests {
				t.Run(fmt.Sprintf("%s/restored=%t/%s", actor.name, restored, test.name), func(t *testing.T) {
					n, client := newShareNavigatorForTest(t, true, actor.id)
					if restored {
						state := shareNavigatorState{
							ShareRoot:       n.shareRoot,
							OwnerRoot:       n.ownerRoot,
							SingleFileShare: n.singleFileShare,
							Share:           n.share,
							Owner:           n.owner,
						}
						n = NewShareNavigator(n.user, client, nil, nil, &setting.DBFS{}, nil).(*shareNavigator)
						require.NoError(t, n.RestoreState(state))
					}

					uri, err := fs.NewUriFromString("cloudreve://test-share@share" + test.path)
					require.NoError(t, err)
					file, err := n.To(context.Background(), uri)
					if test.id == 0 {
						require.ErrorIs(t, err, fs.ErrPathNotExist)
						require.Nil(t, file)
					} else {
						require.NoError(t, err)
						require.Equal(t, test.id, file.ID())
						if file == n.shareRoot {
							list, err := n.Children(context.Background(), file, nil)
							require.NoError(t, err)
							require.True(t, list.SingleFileView)
							require.Len(t, list.Files, 1)
							require.Equal(t, client.shared.ID, list.Files[0].ID())
						}
					}
					require.Zero(t, client.childLookups, "single-file shares must not traverse the owner's directory")
				})
			}
		}
	}
}

func TestShareNavigatorRenamedSingleFile(t *testing.T) {
	n, client := newShareNavigatorForTest(t, true, 2)
	client.shared = &ent.File{ID: 11, Name: "renamed.txt", Type: int(types.FileTypeFile), OwnerID: 1}
	client.children[n.shareRoot.ID()]["shared.txt"] = &ent.File{
		ID: 14, Name: "shared.txt", Type: int(types.FileTypeFile), OwnerID: 1,
	}

	for _, name := range []string{"shared.txt", "renamed.txt"} {
		uri, err := fs.NewUriFromString("cloudreve://test-share@share/" + name)
		require.NoError(t, err)
		file, err := n.To(context.Background(), uri)
		if name == "shared.txt" {
			require.ErrorIs(t, err, fs.ErrPathNotExist)
			require.Nil(t, file)
		} else {
			require.NoError(t, err)
			require.Equal(t, client.shared.ID, file.ID())
			require.Equal(t, name, file.Name())
		}
	}
	require.Zero(t, client.childLookups)
}

func TestShareNavigatorFolderDescendant(t *testing.T) {
	n, client := newShareNavigatorForTest(t, false, 2)
	uri, err := fs.NewUriFromString("cloudreve://test-share@share/private/secret.txt")
	require.NoError(t, err)
	file, err := n.To(context.Background(), uri)
	require.NoError(t, err)
	require.Equal(t, client.children[12]["secret.txt"].ID, file.ID())
	require.Equal(t, 2, client.childLookups)
}
