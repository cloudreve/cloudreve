package explorer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/stretchr/testify/require"
)

func TestBuildListResponseSingleFileViewRedactsParent(t *testing.T) {
	hasher, err := hashid.New("share-parent-test")
	require.NoError(t, err)
	shareURI, err := fs.NewUriFromString("cloudreve://test-share@share/")
	require.NoError(t, err)
	createdAt := time.Date(2026, 9, 1, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	parent := &dbfs.File{
		Model: &ent.File{
			ID:        42,
			Name:      "Project Falcon (confidential)",
			Type:      int(types.FileTypeFolder),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		OwnerModel: &ent.User{ID: 1},
		Path:       [2]*fs.URI{shareURI, shareURI},
	}
	file := &dbfs.File{
		Model: &ent.File{
			ID:   43,
			Name: "plan.pdf",
			Type: int(types.FileTypeFile),
		},
		Parent: parent,
		Path:   [2]*fs.URI{shareURI.Join("plan.pdf"), shareURI.Join("plan.pdf")},
	}

	for _, test := range []struct {
		name           string
		singleFileView bool
		requesterID    int
	}{
		{name: "anonymous single-file recipient", singleFileView: true},
		{name: "signed-in single-file recipient", singleFileView: true, requesterID: 2},
		{name: "folder listing preserves parent", requesterID: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			requester := &ent.User{
				ID: test.requesterID,
				Edges: ent.UserEdges{
					Group: &ent.Group{Permissions: &boolset.BooleanSet{}},
				},
			}
			response := BuildListResponse(context.Background(), requester, parent, &fs.ListFileResult{
				Files:          []fs.File{file},
				Parent:         parent,
				Props:          &fs.NavigatorProps{},
				SingleFileView: test.singleFileView,
			}, hasher)
			data, err := json.Marshal(response)
			require.NoError(t, err)
			var decoded ListResponse
			require.NoError(t, json.Unmarshal(data, &decoded))
			require.Equal(t, test.singleFileView, decoded.SingleFileView)
			require.Len(t, decoded.Files, 1)
			require.Equal(t, "plan.pdf", decoded.Files[0].Name)
			require.Equal(t, hashid.EncodeFileID(hasher, file.ID()), decoded.Files[0].ID)
			require.Equal(t, file.Uri(false).String(), decoded.Files[0].Path)

			if test.singleFileView {
				require.Equal(t, FileResponse{}, decoded.Parent)
			} else {
				require.Equal(t, parent.DisplayName(), decoded.Parent.Name)
				require.Equal(t, hashid.EncodeFileID(hasher, parent.ID()), decoded.Parent.ID)
				require.Equal(t, createdAt, decoded.Parent.CreatedAt)
				require.Equal(t, updatedAt, decoded.Parent.UpdatedAt)
				require.Equal(t, shareURI.String(), decoded.Parent.Path)
			}
		})
	}
}
