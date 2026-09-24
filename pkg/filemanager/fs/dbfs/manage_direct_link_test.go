package dbfs

import (
	"context"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	entuser "github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/stretchr/testify/require"
)

type directLinkFileClient struct {
	inventory.FileClient
	root *ent.File
}

func (c *directLinkFileClient) GetParentFile(_ context.Context, file *ent.File, _ bool) (*ent.File, error) {
	if file.FileChildren == c.root.ID {
		return c.root, nil
	}
	return nil, &ent.NotFoundError{}
}

type directLinkSettingProvider struct {
	setting.Provider
}

func (directLinkSettingProvider) DBFS(context.Context) *setting.DBFS {
	return &setting.DBFS{}
}

func TestGetFileFromDirectLinkChecksOwnerAccess(t *testing.T) {
	allowedGroup := &ent.Group{Settings: &types.GroupSetting{SourceBatchSize: 1}}
	tests := []struct {
		name    string
		status  entuser.Status
		group   *ent.Group
		wantErr bool
	}{
		{name: "active owner with direct link permission", status: entuser.StatusActive, group: allowedGroup},
		{name: "active owner without direct link permission", status: entuser.StatusActive, group: &ent.Group{Settings: &types.GroupSetting{}}, wantErr: true},
		{name: "negative batch size", status: entuser.StatusActive, group: &ent.Group{Settings: &types.GroupSetting{SourceBatchSize: -1}}, wantErr: true},
		{name: "missing group", status: entuser.StatusActive, wantErr: true},
		{name: "missing group settings", status: entuser.StatusActive, group: &ent.Group{}, wantErr: true},
		{name: "manually banned owner", status: entuser.StatusManualBanned, group: allowedGroup, wantErr: true},
		{name: "system banned owner", status: entuser.StatusSysBanned, group: allowedGroup, wantErr: true},
		{name: "inactive owner", status: entuser.StatusInactive, group: allowedGroup, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := &ent.User{ID: 1, Status: tt.status}
			owner.SetGroup(tt.group)
			root := &ent.File{ID: 1, Name: inventory.RootFolderName, OwnerID: owner.ID}
			file := &ent.File{ID: 2, Name: "shared.txt", OwnerID: owner.ID, FileChildren: root.ID}
			file.SetOwner(owner)
			link := &ent.DirectLink{}
			link.SetFile(file)
			dbfs := &DBFS{
				user:          owner,
				fileClient:    &directLinkFileClient{root: root},
				settingClient: directLinkSettingProvider{},
			}

			got, err := dbfs.GetFileFromDirectLink(context.Background(), link)
			if got != nil {
				t.Cleanup(got.(*File).Parent.Recycle)
			}
			if tt.wantErr {
				require.Nil(t, got)
				var appErr serializer.AppError
				require.ErrorAs(t, err, &appErr)
				require.Equal(t, fs.ErrDirectLinkInvalid.Code, appErr.Code)
				require.Equal(t, fs.ErrDirectLinkInvalid.Msg, appErr.Msg)
				return
			}

			require.NoError(t, err)
			require.Same(t, file, got.(*File).Model)
		})
	}
}
