package ytdlp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/downloader"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/samber/lo"
)

const (
	// YtDlpTempFolder is the sub-directory under the node's temp path holding
	// per-task download directories.
	YtDlpTempFolder = "ytdlp"
	defaultBinary   = "yt-dlp"
)

var (
	// [download]  12.3% of ~10.50MiB at 1.23MiB/s ETA 00:08
	progressRe = regexp.MustCompile(`\[download\]\s+([\d.]+)%\s+of\s+~?([\d.]+\s*(?:[KMGTP]i?B|B))\s+at\s+([\d.]+\s*(?:[KMGTP]i?B|B))/s`)
	// [download] 100% of 10.50MiB in 00:08
	completeRe = regexp.MustCompile(`\[download\]\s+100%`)
	sizeRe     = regexp.MustCompile(`^([\d.]+)\s*([KMGTP]i?B|B)$`)
)

type (
	ytdlpClient struct {
		l        logging.Logger
		settings setting.Provider
		options  *types.YtdlpSetting

		mu    sync.Mutex
		procs map[string]*ytdlpProcess
	}

	ytdlpProcess struct {
		cmd    *exec.Cmd
		dir    string
		status *downloader.TaskStatus

		mu      sync.Mutex
		done    chan struct{}
		errTail []string
	}
)

// New creates a yt-dlp based Downloader. yt-dlp is a CLI tool, so the client
// manages spawned processes and parses their progress output.
func New(l logging.Logger, settings setting.Provider, options *types.YtdlpSetting) downloader.Downloader {
	return &ytdlpClient{
		l:        l,
		settings: settings,
		options:  options,
		procs:    map[string]*ytdlpProcess{},
	}
}

func (c *ytdlpClient) binary() string {
	if c.options != nil && c.options.Binary != "" {
		return c.options.Binary
	}
	return defaultBinary
}

func (c *ytdlpClient) tempPath(ctx context.Context) string {
	base := ""
	if c.options != nil {
		base = util.RelativePath(c.options.TempPath)
	}
	if base == "" && c.settings != nil {
		base = util.DataPath(c.settings.TempPath(ctx))
	}
	guid, _ := uuid.NewV4()
	return filepath.Join(base, YtDlpTempFolder, guid.String())
}

// optionArgs converts the settings+per-task option maps into yt-dlp CLI flags.
// Keys are used verbatim: "format" -> "--format <v>", "no_playlist" (bool) ->
// "--no-playlist". This mirrors how aria2 options map to RPC fields.
func optionArgs(options map[string]any) []string {
	keys := lo.Keys(options)
	sort.Strings(keys)

	args := make([]string, 0, len(options)*2)
	for _, k := range keys {
		v := options[k]
		flag := "--" + strings.ReplaceAll(k, "_", "-")
		switch val := v.(type) {
		case bool:
			if val {
				args = append(args, flag)
			}
		case nil:
		case []string:
			for _, item := range val {
				args = append(args, flag, item)
			}
		case []any:
			for _, item := range val {
				args = append(args, flag, fmt.Sprintf("%v", item))
			}
		default:
			args = append(args, flag, fmt.Sprintf("%v", v))
		}
	}
	return args
}

func (c *ytdlpClient) CreateTask(ctx context.Context, url string, options map[string]interface{}) (*downloader.TaskHandle, error) {
	dir := c.tempPath(ctx)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	merged := map[string]any{}
	if c.options != nil {
		for k, v := range c.options.Options {
			merged[k] = v
		}
	}
	for k, v := range options {
		merged[k] = v
	}

	// "output" is a pseudo-key: it replaces the -o template rather than
	// producing a "--output" flag (which yt-dlp does not have).
	output := "%(title)s.%(ext)s"
	if v, ok := merged["output"]; ok {
		output = fmt.Sprintf("%v", v)
		delete(merged, "output")
	}

	args := []string{
		"--newline", "--no-colors",
		"-P", dir,
		"-o", output,
	}
	args = append(args, optionArgs(merged)...)
	// "--" guards against URLs that could be mistaken for flags.
	args = append(args, "--", url)

	c.l.Info("Creating yt-dlp task: %s %s", c.binary(), strings.Join(args, " "))
	// Deliberately not CommandContext: the queue ctx is canceled when each
	// task iteration returns, which would kill the process as soon as the
	// create phase suspends. Lifetime is managed via Cancel/exit instead.
	cmd := exec.Command(c.binary(), args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open yt-dlp stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open yt-dlp stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	p := &ytdlpProcess{
		cmd:  cmd,
		dir:  dir,
		done: make(chan struct{}),
		status: &downloader.TaskStatus{
			Name:     url,
			State:    downloader.StatusDownloading,
			SavePath: filepath.ToSlash(dir),
		},
	}
	go p.run(stdout, stderr)

	c.mu.Lock()
	c.procs[p.id()] = p
	c.mu.Unlock()

	return &downloader.TaskHandle{ID: p.id(), Hash: dir}, nil
}

func (p *ytdlpProcess) id() string { return fmt.Sprintf("%d|%s", p.cmd.Process.Pid, p.dir) }

// run consumes the process output until exit and records terminal state.
func (p *ytdlpProcess) run(stdout, stderr io.Reader) {
	defer close(p.done)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		p.scanProgress(stdout)
	}()
	go func() {
		defer wg.Done()
		p.scanErrors(stderr)
	}()

	err := p.cmd.Wait()
	wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.status.State = downloader.StatusError
		if len(p.errTail) > 0 {
			p.status.ErrorMessage = p.errTail[len(p.errTail)-1]
		} else {
			p.status.ErrorMessage = err.Error()
		}
		return
	}
	p.status.State = downloader.StatusCompleted
	if p.status.Total == 0 {
		p.status.Total = p.status.Downloaded
	}
}

func (p *ytdlpProcess) scanProgress(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if m := progressRe.FindStringSubmatch(line); m != nil {
			p.mu.Lock()
			p.status.Total = parseSize(m[2])
			p.status.DownloadSpeed = parseSize(m[3])
			if p.status.Total > 0 {
				pct, _ := strconv.ParseFloat(m[1], 64)
				p.status.Downloaded = int64(pct / 100 * float64(p.status.Total))
			}
			p.mu.Unlock()
			continue
		}
		if completeRe.MatchString(line) {
			p.mu.Lock()
			p.status.Downloaded = p.status.Total
			p.mu.Unlock()
		}
	}
}

func (p *ytdlpProcess) scanErrors(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		p.mu.Lock()
		p.errTail = append(p.errTail, line)
		if len(p.errTail) > 8 {
			p.errTail = p.errTail[len(p.errTail)-8:]
		}
		p.mu.Unlock()
	}
}

// parseSize converts yt-dlp human sizes ("10.5MiB", "2 GiB") to bytes.
func parseSize(s string) int64 {
	m := sizeRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	num, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	units := map[string]float64{
		"B": 1, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12, "PB": 1e15,
		"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40, "PiB": 1 << 50,
	}
	mult, ok := units[m[2]]
	if !ok {
		return 0
	}
	return int64(num * mult)
}

func (c *ytdlpClient) getProcess(id string) (*ytdlpProcess, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.procs[id]; ok {
		return p, nil
	}
	return nil, downloader.ErrTaskNotFount
}

func (c *ytdlpClient) Info(ctx context.Context, handle *downloader.TaskHandle) (*downloader.TaskStatus, error) {
	p, err := c.getProcess(handle.ID)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	status := *p.status
	terminal := status.State == downloader.StatusCompleted || status.State == downloader.StatusError
	p.mu.Unlock()

	// Enumerate downloaded files for the transfer phase.
	files, err := listDirFiles(p.dir)
	if err != nil {
		return nil, err
	}
	status.Files = files

	if terminal {
		// Terminal status is never queried again (monitor moves to the
		// transfer phase) - release the process entry.
		c.mu.Lock()
		delete(c.procs, handle.ID)
		c.mu.Unlock()
	}
	return &status, nil
}

// listDirFiles returns downloaded files relative to dir, mimicking the aria2
// file list shape the transfer phase consumes.
func listDirFiles(dir string) ([]downloader.TaskFile, error) {
	files := []downloader.TaskFile{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, downloader.TaskFile{
			Index:    len(files),
			Name:     filepath.ToSlash(rel),
			Size:     info.Size(),
			Progress: 1,
			Selected: true,
		})
		return nil
	})
	return files, err
}

func (c *ytdlpClient) Cancel(ctx context.Context, handle *downloader.TaskHandle) error {
	p, err := c.getProcess(handle.ID)
	if err != nil {
		return nil // already gone
	}
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	c.mu.Lock()
	for k, v := range c.procs {
		if v == p {
			delete(c.procs, k)
		}
	}
	c.mu.Unlock()
	return nil
}

// SetFilesToDownload is a no-op: yt-dlp resolves its own file list from the URL.
func (c *ytdlpClient) SetFilesToDownload(ctx context.Context, handle *downloader.TaskHandle, args ...*downloader.SetFileToDownloadArgs) error {
	return nil
}

func (c *ytdlpClient) Test(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, c.binary(), "--version").Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp test failed: %w", err)
	}
	return fmt.Sprintf("yt-dlp %s", strings.TrimSpace(string(out))), nil
}
