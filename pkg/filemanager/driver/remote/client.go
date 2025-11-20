package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/chunk"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/chunk/backoff"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gofrs/uuid"
)

const (
	OverwriteHeader = constants.CrHeaderPrefix + "Overwrite"
	chunkRetrySleep = time.Duration(5) * time.Second
)

// Client to operate uploading to remote slave server
type Client interface {
	// CreateUploadSession creates remote upload session
	CreateUploadSession(ctx context.Context, session *fs.UploadSession, overwrite bool) error
	// GetUploadURL signs an url for uploading file
	GetUploadURL(ctx context.Context, expires time.Time, sessionID string) (string, string, error)
	// Upload uploads file to remote server
	Upload(ctx context.Context, file *fs.UploadRequest) error
	// DeleteUploadSession deletes remote upload session
	DeleteUploadSession(ctx context.Context, sessionID string) error
	// MediaMeta gets media meta from remote server
	MediaMeta(ctx context.Context, src, ext, language string) ([]driver.MediaMeta, error)
	// DeleteFiles deletes files from remote server
	DeleteFiles(ctx context.Context, files ...string) ([]string, error)
	// List lists files from remote server
	List(ctx context.Context, path string, recursive bool) ([]fs.PhysicalObject, error)
}

type DeleteFileRequest struct {
	Files []string `json:"files"`
}

// NewClient creates new Client from given policy
func NewClient(ctx context.Context, policy *ent.StoragePolicy, settings setting.Provider, config conf.ConfigProvider, l logging.Logger) (Client, error) {
	if policy.Edges.Node == nil {
		return nil, fmt.Errorf("remote storage policy %d has no node", policy.ID)
	}

	authInstance := auth.HMACAuth{[]byte(policy.Edges.Node.SlaveKey)}
	serverURL, err := url.Parse(policy.Edges.Node.Server)
	if err != nil {
		return nil, err
	}

	base, _ := url.Parse(constants.APIPrefixSlave)

	return &remoteClient{
		policy:       policy,
		authInstance: authInstance,
		httpClient: request.NewClient(
			config,
			request.WithEndpoint(serverURL.ResolveReference(base).String()),
			request.WithCredential(authInstance, int64(settings.SlaveRequestSignTTL(ctx))),
			request.WithSlaveMeta(policy.Edges.Node.ID),
			request.WithMasterMeta(settings.SiteBasic(ctx).ID, settings.SiteURL(setting.UseFirstSiteUrl(ctx)).String()),
			request.WithCorrelationID(),
		),
		settings: settings,
		l:        l,
	}, nil
}

type remoteClient struct {
	policy       *ent.StoragePolicy
	authInstance auth.Auth
	httpClient   request.Client
	settings     setting.Provider
	l            logging.Logger
}

func (c *remoteClient) Upload(ctx context.Context, file *fs.UploadRequest) error {
	ttl := c.settings.UploadSessionTTL(ctx)
	session := &fs.UploadSession{
		Props:  file.Props.Copy(),
		Policy: c.policy,
	}
	session.Props.UploadSessionID = uuid.Must(uuid.NewV4()).String()
	session.Props.ExpireAt = time.Now().Add(ttl)

	// Create upload session
	overwrite := file.Mode&fs.ModeOverwrite == fs.ModeOverwrite
	if err := c.CreateUploadSession(ctx, session, overwrite); err != nil {
		return fmt.Errorf("failed to create upload session: %w", err)
	}

	// Initial chunk groups
	chunks := chunk.NewChunkGroup(file, c.policy.Settings.ChunkSize, &backoff.ConstantBackoff{
		Max:   c.settings.ChunkRetryLimit(ctx),
		Sleep: chunkRetrySleep,
	}, c.settings.UseChunkBuffer(ctx), c.l, c.settings.TempPath(ctx))

	uploadFunc := func(current *chunk.ChunkGroup, content io.Reader) error {
		return c.uploadChunk(ctx, session.Props.UploadSessionID, current.Index(), content, overwrite, current.Length())
	}

	// upload chunks
	for chunks.Next() {
		if err := chunks.Process(uploadFunc); err != nil {
			if err := c.DeleteUploadSession(ctx, session.Props.UploadSessionID); err != nil {
				c.l.Warning("failed to delete upload session: %s", err)
			}

			return fmt.Errorf("failed to upload chunk #%d: %w", chunks.Index(), err)
		}
	}

	return nil
}

func (c *remoteClient) DeleteUploadSession(ctx context.Context, sessionID string) error {
	resp, err := c.httpClient.Request(
		"DELETE",
		"upload/"+sessionID,
		nil,
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (c *remoteClient) DeleteFiles(ctx context.Context, files ...string) ([]string, error) {
	req := &DeleteFileRequest{
		Files: files,
	}

	reqStr, err := json.Marshal(req)
	if err != nil {
		return files, fmt.Errorf("failed to marshal delete request: %w", err)
	}

	resp, err := c.httpClient.Request(
		"DELETE",
		"file",
		bytes.NewReader(reqStr),
		request.WithContext(ctx),
		request.WithLogger(logging.FromContext(ctx)),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return files, err
	}

	if resp.Code != 0 {
		var failed []string
		failed = files
		if resp.Code == serializer.CodeNotFullySuccess {
			resp.GobDecode(&failed)
		}
		return failed, fmt.Errorf(resp.Error)
	}

	return nil, nil
}

func (c *remoteClient) MediaMeta(ctx context.Context, src, ext, language string) ([]driver.MediaMeta, error) {
	resp, err := c.httpClient.Request(
		http.MethodGet,
		routes.SlaveMediaMetaRoute(src, ext, language),
		nil,
		request.WithContext(ctx),
		request.WithLogger(c.l),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf(resp.Error)
	}

	var metas []driver.MediaMeta
	resp.GobDecode(&metas)
	return metas, nil
}

func (c *remoteClient) CreateUploadSession(ctx context.Context, session *fs.UploadSession, overwrite bool) error {
	reqBodyEncoded, err := json.Marshal(map[string]interface{}{
		"session":   session,
		"overwrite": overwrite,
	})
	if err != nil {
		return err
	}

	bodyReader := strings.NewReader(string(reqBodyEncoded))
	resp, err := c.httpClient.Request(
		"PUT",
		"upload",
		bodyReader,
		request.WithContext(ctx),
		request.WithLogger(c.l),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}

func (c *remoteClient) List(ctx context.Context, path string, recursive bool) ([]fs.PhysicalObject, error) {
	resp, err := c.httpClient.Request(
		http.MethodGet,
		routes.SlaveFileListRoute(path, recursive),
		nil,
		request.WithContext(ctx),
		request.WithLogger(c.l),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, serializer.NewErrorFromResponse(resp)
	}

	var objects []fs.PhysicalObject
	resp.GobDecode(&objects)
	return objects, nil

}

func (c *remoteClient) GetUploadURL(ctx context.Context, expires time.Time, sessionID string) (string, string, error) {
	base, err := url.Parse(c.policy.Edges.Node.Server)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", routes.SlaveUploadUrl(base, sessionID).String(), nil)
	if err != nil {
		return "", "", err
	}

	req = auth.SignRequest(ctx, c.authInstance, req, &expires)
	return req.URL.String(), req.Header["Authorization"][0], nil
}

func (c *remoteClient) uploadChunk(ctx context.Context, sessionID string, index int, chunk io.Reader, overwrite bool, size int64) error {
	resp, err := c.httpClient.Request(
		"POST",
		fmt.Sprintf("upload/%s?chunk=%d", sessionID, index),
		chunk,
		request.WithContext(ctx),
		request.WithTimeout(time.Duration(0)),
		request.WithContentLength(size),
		request.WithHeader(map[string][]string{OverwriteHeader: {fmt.Sprintf("%t", overwrite)}}),
	).CheckHTTPResponse(200).DecodeResponse()
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return serializer.NewErrorFromResponse(resp)
	}

	return nil
}
