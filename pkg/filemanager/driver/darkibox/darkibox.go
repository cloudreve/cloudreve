package darkibox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
)

const apiBase = "https://darkibox.com/api"

// Driver implements driver.Handler for darkibox.com video hosting.
type Driver struct {
	policy     *ent.StoragePolicy
	apiKey     string
	httpClient request.Client
	l          logging.Logger
}

var features = &boolset.BooleanSet{}

func init() {
	boolset.Set(driver.HandlerCapabilityProxyRequired, true, features)
}

// New creates a new Darkibox driver.
func New(_ context.Context, policy *ent.StoragePolicy, _ conf.ConfigProvider, l logging.Logger) (*Driver, error) {
	return &Driver{
		policy:     policy,
		apiKey:     policy.AccessKey,
		httpClient: request.NewClientDeprecated(request.WithLogger(l)),
		l:          l,
	}, nil
}

// apiResponse is the generic envelope returned by all Darkibox API calls.
type apiResponse struct {
	Status  int             `json:"status"`
	Msg     string          `json:"msg"`
	Result  json.RawMessage `json:"result"`
	ServerTime string       `json:"server_time"`
}

// ── file/list result types ──────────────────────────────────────────

type fileListResult struct {
	Results int             `json:"results"`
	Files   []fileListEntry `json:"files"`
}

type fileListEntry struct {
	FileCode string `json:"file_code"`
	Title    string `json:"title"`
	Name     string `json:"name,omitempty"`
	FldID    int    `json:"fld_id"`
	Size     int64  `json:"size"`
	Uploaded string `json:"uploaded"`
}

// ── folder/list result types ────────────────────────────────────────

type folderListResult struct {
	Folders []folderEntry `json:"folders"`
}

type folderEntry struct {
	FldID int    `json:"fld_id"`
	Name  string `json:"name"`
}

// ── file/direct_link result ─────────────────────────────────────────

type directLinkResult struct {
	Versions []directLinkVersion `json:"versions"`
}

type directLinkVersion struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ── upload/server result ────────────────────────────────────────────

type uploadServerResult struct {
	URL string `json:"url"`
}

// ── folder/create result ────────────────────────────────────────────

type folderCreateResult struct {
	FldID int `json:"fld_id"`
}

// ── API helpers ─────────────────────────────────────────────────────

// apiGet performs a GET request against the Darkibox API and decodes the
// JSON envelope. The raw `result` field is returned for the caller to
// unmarshal into the appropriate type.
func (d *Driver) apiGet(ctx context.Context, endpoint string, params map[string]string) (*apiResponse, error) {
	u := apiBase + endpoint + "?key=" + d.apiKey
	for k, v := range params {
		u += "&" + k + "=" + v
	}

	resp := d.httpClient.Request("GET", u, nil, request.WithContext(ctx))
	body, err := resp.GetResponse()
	if err != nil {
		return nil, fmt.Errorf("darkibox: request %s failed: %w", endpoint, err)
	}

	var ar apiResponse
	if err := json.Unmarshal([]byte(body), &ar); err != nil {
		return nil, fmt.Errorf("darkibox: decode response for %s: %w", endpoint, err)
	}
	if ar.Status != 200 {
		return nil, fmt.Errorf("darkibox: %s returned status %d: %s", endpoint, ar.Status, ar.Msg)
	}
	return &ar, nil
}

// ── Handler interface ───────────────────────────────────────────────

// Put uploads a file to Darkibox via the two-step upload flow:
// 1. GET /api/upload/server to obtain the upload URL.
// 2. POST multipart form to that URL with key, file, and fld_id.
//
// The file is stored under the folder derived from the save path. Folders
// along the path are created automatically.
func (d *Driver) Put(ctx context.Context, file *fs.UploadRequest) error {
	defer file.Close()

	// Resolve (or create) the target folder.
	dir := path.Dir(file.Props.SavePath)
	fldID, err := d.ensureFolder(ctx, dir)
	if err != nil {
		return fmt.Errorf("darkibox put: resolve folder: %w", err)
	}

	// Step 1: obtain the upload server URL.
	ar, err := d.apiGet(ctx, "/upload/server", nil)
	if err != nil {
		return err
	}
	var srv uploadServerResult
	if err := json.Unmarshal(ar.Result, &srv); err != nil {
		return fmt.Errorf("darkibox: decode upload server: %w", err)
	}
	if srv.URL == "" {
		return errors.New("darkibox: empty upload server URL")
	}

	// Step 2: multipart upload.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		_ = writer.WriteField("key", d.apiKey)
		_ = writer.WriteField("fld_id", strconv.Itoa(fldID))

		part, err := writer.CreateFormFile("file", path.Base(file.Props.SavePath))
		if err != nil {
			errCh <- err
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- err
			return
		}
		errCh <- writer.Close()
	}()

	resp := d.httpClient.Request("POST", srv.URL, pr,
		request.WithContext(ctx),
		request.WithHeader(http.Header{
			"Content-Type": {writer.FormDataContentType()},
		}),
	)

	// Wait for the writer goroutine to finish.
	if writeErr := <-errCh; writeErr != nil {
		return fmt.Errorf("darkibox put: write multipart: %w", writeErr)
	}

	body, err := resp.GetResponse()
	if err != nil {
		return fmt.Errorf("darkibox put: upload request: %w", err)
	}

	var uploadResp apiResponse
	if err := json.Unmarshal([]byte(body), &uploadResp); err != nil {
		return fmt.Errorf("darkibox put: decode upload response: %w", err)
	}
	if uploadResp.Status != 200 {
		return fmt.Errorf("darkibox put: upload returned status %d: %s", uploadResp.Status, uploadResp.Msg)
	}

	return nil
}

// Delete deletes files by their source paths (which are darkibox file codes).
func (d *Driver) Delete(ctx context.Context, files ...string) ([]string, error) {
	var failed []string
	var lastErr error

	for _, f := range files {
		code := fileCodeFromSource(f)
		_, err := d.apiGet(ctx, "/file/delete", map[string]string{
			"file_code": code,
		})
		if err != nil {
			failed = append(failed, f)
			lastErr = err
		}
	}

	return failed, lastErr
}

// Source returns a direct download URL for the given entity.
func (d *Driver) Source(ctx context.Context, e fs.Entity, _ *driver.GetSourceArgs) (string, error) {
	code := fileCodeFromSource(e.Source())
	ar, err := d.apiGet(ctx, "/file/direct_link", map[string]string{
		"file_code": code,
	})
	if err != nil {
		return "", err
	}

	var dl directLinkResult
	if err := json.Unmarshal(ar.Result, &dl); err != nil {
		return "", fmt.Errorf("darkibox: decode direct_link: %w", err)
	}

	// Pick the original ("o") version, fall back to first available.
	for _, v := range dl.Versions {
		if v.Name == "o" {
			return v.URL, nil
		}
	}
	if len(dl.Versions) > 0 {
		return dl.Versions[0].URL, nil
	}

	return "", errors.New("darkibox: no direct link version available")
}

// List lists files and folders under the given base path.
func (d *Driver) List(ctx context.Context, base string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	base = strings.TrimPrefix(base, "/")

	fldID, err := d.resolveFolderID(ctx, base)
	if err != nil {
		return nil, err
	}

	return d.listFolder(ctx, fldID, base, onProgress, recursive)
}

func (d *Driver) listFolder(ctx context.Context, fldID int, prefix string, onProgress driver.ListProgressFunc, recursive bool) ([]fs.PhysicalObject, error) {
	var objects []fs.PhysicalObject

	// List sub-folders.
	ar, err := d.apiGet(ctx, "/folder/list", map[string]string{
		"fld_id": strconv.Itoa(fldID),
	})
	if err != nil {
		return nil, err
	}

	var fl folderListResult
	if err := json.Unmarshal(ar.Result, &fl); err != nil {
		return nil, fmt.Errorf("darkibox: decode folder list: %w", err)
	}

	for _, folder := range fl.Folders {
		rel := folder.Name
		if prefix != "" {
			rel = prefix + "/" + folder.Name
		}
		objects = append(objects, fs.PhysicalObject{
			Name:         folder.Name,
			RelativePath: rel,
			Source:       rel,
			Size:         0,
			IsDir:        true,
			LastModify:   time.Now(),
		})
		onProgress(1)

		if recursive {
			children, err := d.listFolder(ctx, folder.FldID, rel, onProgress, true)
			if err != nil {
				return nil, err
			}
			objects = append(objects, children...)
		}
	}

	// List files (paginated).
	page := 1
	for {
		far, err := d.apiGet(ctx, "/file/list", map[string]string{
			"fld_id":   strconv.Itoa(fldID),
			"per_page": "200",
			"page":     strconv.Itoa(page),
		})
		if err != nil {
			return nil, err
		}

		var flr fileListResult
		if err := json.Unmarshal(far.Result, &flr); err != nil {
			return nil, fmt.Errorf("darkibox: decode file list: %w", err)
		}

		for _, f := range flr.Files {
			name := f.Title
			if name == "" {
				name = f.Name
			}
			rel := name
			if prefix != "" {
				rel = prefix + "/" + name
			}

			uploaded, _ := time.Parse("2006-01-02 15:04:05", f.Uploaded)

			objects = append(objects, fs.PhysicalObject{
				Name:         name,
				RelativePath: rel,
				Source:       f.FileCode,
				Size:         f.Size,
				IsDir:        false,
				LastModify:   uploaded,
			})
			onProgress(1)
		}

		if len(flr.Files) < 200 {
			break
		}
		page++
	}

	return objects, nil
}

// Token returns upload credentials. For Darkibox the actual upload happens
// server-side via Put, so we return a minimal credential pointing at the
// Cloudreve relay endpoint.
func (d *Driver) Token(ctx context.Context, uploadSession *fs.UploadSession, file *fs.UploadRequest) (*fs.UploadCredential, error) {
	return &fs.UploadCredential{
		SessionID: uploadSession.Props.UploadSessionID,
		Expires:   uploadSession.Props.ExpireAt.Unix(),
	}, nil
}

// CancelToken is a no-op for Darkibox.
func (d *Driver) CancelToken(_ context.Context, _ *fs.UploadSession) error {
	return nil
}

// CompleteUpload is a no-op for Darkibox (upload is atomic via Put).
func (d *Driver) CompleteUpload(_ context.Context, _ *fs.UploadSession) error {
	return nil
}

// Thumb is not supported by Darkibox.
func (d *Driver) Thumb(_ context.Context, _ *time.Time, _ string, _ fs.Entity) (string, error) {
	return "", errors.New("not implemented")
}

// Open is not supported (no local filesystem access).
func (d *Driver) Open(_ context.Context, _ string) (*os.File, error) {
	return nil, errors.New("not implemented")
}

// LocalPath is not supported.
func (d *Driver) LocalPath(_ context.Context, _ string) string {
	return ""
}

// Capabilities returns the driver capabilities.
func (d *Driver) Capabilities() *driver.Capabilities {
	return &driver.Capabilities{
		StaticFeatures: features,
	}
}

// MediaMeta is not supported.
func (d *Driver) MediaMeta(_ context.Context, _, _, _ string) ([]driver.MediaMeta, error) {
	return nil, nil
}

// ── Folder helpers ──────────────────────────────────────────────────

// resolveFolderID walks the path components starting from root (fld_id=0)
// and returns the fld_id of the deepest folder. Returns 0 for the root.
func (d *Driver) resolveFolderID(ctx context.Context, p string) (int, error) {
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return 0, nil
	}

	parts := strings.Split(p, "/")
	currentID := 0

	for _, part := range parts {
		ar, err := d.apiGet(ctx, "/folder/list", map[string]string{
			"fld_id": strconv.Itoa(currentID),
		})
		if err != nil {
			return 0, err
		}

		var fl folderListResult
		if err := json.Unmarshal(ar.Result, &fl); err != nil {
			return 0, err
		}

		found := false
		for _, folder := range fl.Folders {
			if folder.Name == part {
				currentID = folder.FldID
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("darkibox: folder %q not found under fld_id=%d", part, currentID)
		}
	}

	return currentID, nil
}

// ensureFolder is like resolveFolderID but creates missing folders along
// the path.
func (d *Driver) ensureFolder(ctx context.Context, p string) (int, error) {
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return 0, nil
	}

	parts := strings.Split(p, "/")
	currentID := 0

	for _, part := range parts {
		ar, err := d.apiGet(ctx, "/folder/list", map[string]string{
			"fld_id": strconv.Itoa(currentID),
		})
		if err != nil {
			return 0, err
		}

		var fl folderListResult
		if err := json.Unmarshal(ar.Result, &fl); err != nil {
			return 0, err
		}

		found := false
		for _, folder := range fl.Folders {
			if folder.Name == part {
				currentID = folder.FldID
				found = true
				break
			}
		}
		if !found {
			// Create the folder.
			car, err := d.apiGet(ctx, "/folder/create", map[string]string{
				"name":      part,
				"parent_id": strconv.Itoa(currentID),
			})
			if err != nil {
				return 0, fmt.Errorf("darkibox: create folder %q: %w", part, err)
			}

			var cr folderCreateResult
			if err := json.Unmarshal(car.Result, &cr); err != nil {
				return 0, fmt.Errorf("darkibox: decode folder create: %w", err)
			}
			currentID = cr.FldID
		}
	}

	return currentID, nil
}

// fileCodeFromSource extracts the file code from a source string.
// The source may be a plain file code or a path ending with the code.
func fileCodeFromSource(source string) string {
	source = strings.TrimSpace(source)
	if idx := strings.LastIndex(source, "/"); idx >= 0 {
		return source[idx+1:]
	}
	return source
}
