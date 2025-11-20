package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
)

const (
	apiPrefix       = "/api/v2"
	successResponse = "Ok."
	crTagPrefix     = "cr-"

	downloadPrioritySkip     = 0
	downloadPriorityDownload = 1
)

var (
	downloadOptionFormatTypes = map[string]string{
		"cookie":             "%s",
		"skip_checking":      "%s",
		"root_folder":        "%s",
		"rename":             "%s",
		"upLimit":            "%.0f",
		"dlLimit":            "%.0f",
		"ratioLimit":         "%f",
		"seedingTimeLimit":   "%.0f",
		"autoTMM":            "%t",
		"sequentialDownload": "%s",
		"firstLastPiecePrio": "%t",
	}
)

type qbittorrentClient struct {
	c        request.Client
	settings setting.Provider
	l        logging.Logger
	options  *types.QBittorrentSetting
}

func NewClient(l logging.Logger, c request.Client, setting setting.Provider, options *types.QBittorrentSetting) (downloader.Downloader, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	server, err := url.Parse(options.Server)
	if err != nil {
		return nil, fmt.Errorf("invalid qbittorrent server URL: %w", err)
	}

	base, _ := url.Parse(apiPrefix)
	c.Apply(
		request.WithCookieJar(jar),
		request.WithLogger(l),
		request.WithEndpoint(options.Server),
		request.WithEndpoint(server.ResolveReference(base).String()),
	)
	return &qbittorrentClient{c: c, options: options, l: l, settings: setting}, nil
}

func (c *qbittorrentClient) SetFilesToDownload(ctx context.Context, handle *downloader.TaskHandle, args ...*downloader.SetFileToDownloadArgs) error {
	downloadId := make([]int, 0, len(args))
	skipId := make([]int, 0, len(args))
	for _, arg := range args {
		if arg.Download {
			downloadId = append(downloadId, arg.Index)
		} else {
			skipId = append(skipId, arg.Index)
		}
	}

	if len(downloadId) > 0 {
		if err := c.setFilePriority(ctx, handle.Hash, downloadPriorityDownload, downloadId...); err != nil {
			return fmt.Errorf("failed to set file priority to download: %w", err)
		}
	}

	if len(skipId) > 0 {
		if err := c.setFilePriority(ctx, handle.Hash, downloadPrioritySkip, skipId...); err != nil {
			return fmt.Errorf("failed to set file priority to skip: %w", err)
		}
	}

	return nil
}

func (c *qbittorrentClient) Cancel(ctx context.Context, handle *downloader.TaskHandle) error {
	buffer := bytes.Buffer{}
	formWriter := multipart.NewWriter(&buffer)
	_ = formWriter.WriteField("hashes", handle.Hash)
	_ = formWriter.WriteField("deleteFiles", "true")

	headers := http.Header{
		"Content-Type": []string{formWriter.FormDataContentType()},
	}

	_, err := c.request(ctx, http.MethodPost, "torrents/delete", buffer.String(), &headers)
	if err != nil {
		return fmt.Errorf("failed to cancel task with hash %q: %w", handle.Hash, err)
	}

	// Delete tags
	buffer = bytes.Buffer{}
	formWriter = multipart.NewWriter(&buffer)
	_ = formWriter.WriteField("tags", crTagPrefix+handle.ID)

	headers = http.Header{
		"Content-Type": []string{formWriter.FormDataContentType()},
	}

	_, err = c.request(ctx, http.MethodPost, "torrents/deleteTags", buffer.String(), &headers)
	if err != nil {
		return fmt.Errorf("failed to delete tag with id %q: %w", handle.ID, err)
	}

	return nil
}

func (c *qbittorrentClient) Info(ctx context.Context, handle *downloader.TaskHandle) (*downloader.TaskStatus, error) {
	buffer := bytes.Buffer{}
	formWriter := multipart.NewWriter(&buffer)
	_ = formWriter.WriteField("tag", crTagPrefix+handle.ID)

	headers := http.Header{
		"Content-Type": []string{formWriter.FormDataContentType()},
	}

	// Get task info
	resp, err := c.request(ctx, http.MethodPost, "torrents/info", buffer.String(), &headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get task info with tag %q: %w", crTagPrefix+handle.ID, err)
	}

	var torrents []Torrent
	if err := json.Unmarshal([]byte(resp), &torrents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal info response: %w", err)
	}

	if len(torrents) == 0 {
		return nil, fmt.Errorf("no torrent under tag %q: %w", crTagPrefix+handle.ID, downloader.ErrTaskNotFount)
	}

	// Get file info
	buffer = bytes.Buffer{}
	formWriter = multipart.NewWriter(&buffer)
	_ = formWriter.WriteField("hash", torrents[0].Hash)
	headers = http.Header{
		"Content-Type": []string{formWriter.FormDataContentType()},
	}

	resp, err = c.request(ctx, http.MethodPost, "torrents/files", buffer.String(), &headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get torrent files with hash %q: %w", torrents[0].Hash, err)
	}

	var files []File
	if err := json.Unmarshal([]byte(resp), &files); err != nil {
		return nil, fmt.Errorf("failed to unmarshal files response: %w", err)
	}

	// Get piece status
	resp, err = c.request(ctx, http.MethodPost, "torrents/pieceStates", buffer.String(), &headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get torrent pieceStates with hash %q: %w", torrents[0].Hash, err)
	}

	var pieceStates []int
	if err := json.Unmarshal([]byte(resp), &pieceStates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pieceStates response: %w", err)
	}

	// Combining and converting all info
	state := downloader.StatusDownloading
	switch torrents[0].State {
	case "downloading", "pausedDL", "allocating", "metaDL", "queuedDL", "stalledDL", "checkingDL", "forcedDL", "checkingResumeData", "moving", "forcedMetaDL":
		state = downloader.StatusDownloading
	case "uploading", "queuedUP", "stalledUP", "checkingUP", "forcedUP":
		state = downloader.StatusSeeding
	case "pausedUP", "stoppedUP":
		state = downloader.StatusCompleted
	case "error", "missingFiles":
		state = downloader.StatusError
	default:
		state = downloader.StatusUnknown
	}
	status := &downloader.TaskStatus{
		Name:          torrents[0].Name,
		Total:         torrents[0].Size,
		Downloaded:    torrents[0].Completed,
		DownloadSpeed: torrents[0].Dlspeed,
		Uploaded:      torrents[0].Uploaded,
		UploadSpeed:   torrents[0].Upspeed,
		SavePath:      filepath.ToSlash(torrents[0].SavePath),
		State:         state,
		Hash:          torrents[0].Hash,
		Files: lo.Map(files, func(item File, index int) downloader.TaskFile {
			return downloader.TaskFile{
				Index:    item.Index,
				Name:     filepath.ToSlash(item.Name),
				Size:     item.Size,
				Progress: item.Progress,
				Selected: item.Priority > 0,
			}
		}),
	}

	if handle.Hash != torrents[0].Hash {
		handle.Hash = torrents[0].Hash
		status.FollowedBy = handle
	}

	// Convert piece states to hex bytes array, The highest bit corresponds to the piece at index 0.
	status.NumPieces = len(pieceStates)
	pieces := make([]byte, 0, len(pieceStates)/8+1)
	for i := 0; i < len(pieceStates); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if i+j >= len(pieceStates) {
				break
			}
			pieceStatus := 0
			if pieceStates[i+j] == 2 {
				pieceStatus = 1
			}
			b |= byte(pieceStatus) << uint(7-j)
		}
		pieces = append(pieces, b)
	}
	status.Pieces = pieces

	return status, nil
}

func (c *qbittorrentClient) CreateTask(ctx context.Context, url string, options map[string]interface{}) (*downloader.TaskHandle, error) {
	guid, _ := uuid.NewV4()

	// Generate a unique path for the task
	base := util.RelativePath(c.options.TempPath)
	if c.options.TempPath == "" {
		base = util.DataPath(c.settings.TempPath(ctx))
	}
	path := filepath.Join(
		base,
		"qbittorrent",
		guid.String(),
	)
	c.l.Info("Creating QBitTorrent task with url %q saving to %q...", url, path)

	buffer := bytes.Buffer{}
	formWriter := multipart.NewWriter(&buffer)
	_ = formWriter.WriteField("urls", url)
	_ = formWriter.WriteField("savepath", path)
	_ = formWriter.WriteField("tags", crTagPrefix+guid.String())

	// Apply global options
	for k, v := range c.options.Options {
		if _, ok := downloadOptionFormatTypes[k]; ok {
			_ = formWriter.WriteField(k, fmt.Sprintf(downloadOptionFormatTypes[k], v))
		}
	}

	// Apply group options
	for k, v := range options {
		if _, ok := downloadOptionFormatTypes[k]; ok {
			_ = formWriter.WriteField(k, fmt.Sprintf(downloadOptionFormatTypes[k], v))
		}
	}

	// Send request
	headers := http.Header{
		"Content-Type": []string{formWriter.FormDataContentType()},
	}

	resp, err := c.request(ctx, http.MethodPost, "torrents/add", buffer.String(), &headers)
	if err != nil {
		return nil, fmt.Errorf("create task qbittorrent failed: %w", err)
	}

	if resp != successResponse {
		return nil, fmt.Errorf("create task qbittorrent failed: %s", resp)
	}

	return &downloader.TaskHandle{
		ID: guid.String(),
	}, nil
}

func (c *qbittorrentClient) setFilePriority(ctx context.Context, hash string, priority int, id ...int) error {
	buffer := bytes.Buffer{}
	formWriter := multipart.NewWriter(&buffer)
	_ = formWriter.WriteField("hash", hash)
	_ = formWriter.WriteField("id", strings.Join(
		lo.Map(id, func(item int, index int) string {
			return fmt.Sprintf("%d", item)
		}), "|"))
	_ = formWriter.WriteField("priority", fmt.Sprintf("%d", priority))

	headers := http.Header{
		"Content-Type": []string{formWriter.FormDataContentType()},
	}

	_, err := c.request(ctx, http.MethodPost, "torrents/filePrio", buffer.String(), &headers)
	if err != nil {
		return fmt.Errorf("failed to set file priority: %w", err)
	}

	return nil
}

func (c *qbittorrentClient) Test(ctx context.Context) (string, error) {
	res, err := c.request(ctx, http.MethodGet, "app/version", "", nil)
	if err != nil {
		return "", fmt.Errorf("test qbittorrent failed: %w", err)
	}

	return res, nil
}

func (c *qbittorrentClient) login(ctx context.Context) error {
	form := url.Values{}
	form.Add("username", c.options.User)
	form.Add("password", c.options.Password)
	res, err := c.c.Request(http.MethodPost, "auth/login",
		strings.NewReader(form.Encode()),
		request.WithContext(ctx),
		request.WithHeader(http.Header{
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		}),
	).CheckHTTPResponse(http.StatusOK).GetResponse()
	if err != nil {
		return fmt.Errorf("login failed with unexpected status code: %w", err)
	}

	if res != successResponse {
		return fmt.Errorf("login failed with response: %s, possibly inccorrect credential is provided", res)
	}

	return nil
}

func (c *qbittorrentClient) request(ctx context.Context, method, path string, body string, headers *http.Header) (string, error) {
	opts := []request.Option{
		request.WithContext(ctx),
	}

	if headers != nil {
		opts = append(opts, request.WithHeader(*headers))
	}

	res := c.c.Request(method, path, strings.NewReader(body), opts...)

	if res.Err != nil {
		return "", fmt.Errorf("send request failed: %w", res.Err)
	}

	switch res.Response.StatusCode {
	case http.StatusForbidden:
		c.l.Info("QBittorrent cookie expired, sending login request...")
		if err := c.login(ctx); err != nil {
			return "", fmt.Errorf("login failed: %w", err)
		}

		return c.request(ctx, method, path, body, headers)

	case http.StatusOK:
		respContent, err := res.GetResponse()
		if err != nil {
			return "", fmt.Errorf("failed reading response: %w", err)
		}

		return respContent, nil
	case http.StatusUnsupportedMediaType:
		return "", fmt.Errorf("invalid torrent file")
	default:
		content, _ := res.GetResponse()
		return "", fmt.Errorf("unexpected status code: %d, content: %s", res.Response.StatusCode, content)
	}
}
