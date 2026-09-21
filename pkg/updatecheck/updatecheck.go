package updatecheck

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
)

const (
	// ReleaseRepo is the GitHub repository releases are fetched from.
	ReleaseRepo     = "Dvorinka/cloudreve"
	releaseAPI      = "https://api.github.com/repos/" + ReleaseRepo + "/releases"
	releaseTTL      = 6 * time.Hour
	cacheKey        = "server_update_latest"
	checkTimeout    = 15 * time.Second
	downloadTimeout = 10 * time.Minute
)

// serverTag matches server release tags (v4.19.2, v4.19.2-rc1) and rejects the
// desktop-v* / android-v* tag families living in the same repository.
var serverTag = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

func init() {
	// Required for *ReleaseInfo to survive a round trip through the
	// gob-encoded cache driver (Redis).
	gob.Register(&ReleaseInfo{})
}

// ReleaseInfo describes the newest upstream release.
type ReleaseInfo struct {
	Version     string    `json:"version"`
	URL         string    `json:"url"`
	Notes       string    `json:"notes"`
	PublishedAt time.Time `json:"published_at"`
	Newer       bool      `json:"newer"`
	Current     string    `json:"current"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// LatestRelease queries GitHub for the newest server release. Results are
// cached in the KV store for releaseTTL so admin page loads do not hammer the
// API. A nil kv skips the cache. Failures return an error.
func LatestRelease(ctx context.Context, kv cache.Driver) (*ReleaseInfo, error) {
	if kv != nil {
		if cached, ok := kv.Get(cacheKey); ok {
			if rel, ok := cached.(*ReleaseInfo); ok {
				return rel, nil
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cloudreve-update-check")

	client := &http.Client{Timeout: checkTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release check failed: %s", resp.Status)
	}

	var releases []ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&releases); err != nil {
		return nil, err
	}

	// The repo hosts server (v*), desktop (desktop-v*) and Android (android-v*)
	// tag families; the list is newest-first so the first matching server tag
	// is the latest server release.
	var gh *ghRelease
	for i := range releases {
		if serverTag.MatchString(releases[i].TagName) {
			gh = &releases[i]
			break
		}
	}
	if gh == nil {
		return nil, fmt.Errorf("no server release found")
	}

	rel := &ReleaseInfo{
		Version:     strings.TrimPrefix(gh.TagName, "v"),
		URL:         gh.HTMLURL,
		Notes:       gh.Body,
		PublishedAt: gh.PublishedAt,
		Current:     strings.TrimPrefix(constants.BackendVersion, "v"),
	}
	rel.Newer = IsNewer(rel.Current, rel.Version)

	if kv != nil {
		_ = kv.Set(cacheKey, rel, int(releaseTTL.Seconds()))
	}
	return rel, nil
}

// IsNewer reports whether latest is a strictly newer version than current.
// Versions are compared numerically per component; pre-release suffixes make a
// version older than the same version without one (4.19.2-rc1 < 4.19.2).
func IsNewer(current, latest string) bool {
	cur := parseVersion(current)
	lat := parseVersion(latest)
	for i := 0; i < 3; i++ {
		if lat.nums[i] != cur.nums[i] {
			return lat.nums[i] > cur.nums[i]
		}
	}
	// equal numeric parts: a pre-release is older than the release
	if cur.pre != lat.pre {
		return lat.pre == ""
	}
	return false
}

type semv struct {
	nums [3]int
	pre  string
}

func parseVersion(v string) semv {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var out semv
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		out.pre = v[i+1:]
		v = v[:i]
	}
	for i, part := range strings.Split(v, ".") {
		if i > 2 {
			break
		}
		n, _ := strconv.Atoi(part)
		out.nums[i] = n
	}
	return out
}

// InContainer reports whether the process runs inside a container where
// self-updating the binary is meaningless (the image owns the binary).
func InContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	return os.Getenv("container") != ""
}

// SelfUpdateSupported reports whether this build can self-update in place.
func SelfUpdateSupported() (bool, string) {
	if InContainer() {
		return false, "container"
	}
	if _, err := os.Executable(); err != nil {
		return false, "no-executable"
	}
	return true, ""
}

// platformAssetName maps the current platform to a release asset name.
func platformAssetName(tag string) (string, error) {
	arch := runtime.GOARCH
	if arch == "arm" {
		// GOARM is not exposed at runtime; derive it from uname on Linux.
		arch = "armv7"
		if runtime.GOOS == "linux" {
			if out, err := unameMachine(); err == nil {
				switch {
				case strings.HasPrefix(out, "armv5"):
					arch = "armv5"
				case strings.HasPrefix(out, "armv6"):
					arch = "armv6"
				default:
					arch = "armv7"
				}
			}
		}
	}

	supported := map[string]bool{
		"amd64": true, "arm64": true, "loong64": true,
		"armv5": true, "armv6": true, "armv7": true,
	}
	if !supported[arch] {
		return "", fmt.Errorf("unsupported architecture for self-update: %s", runtime.GOARCH)
	}

	switch runtime.GOOS {
	case "linux", "darwin", "freebsd":
		return fmt.Sprintf("cloudreve_%s_%s_%s.tar.gz", tag, runtime.GOOS, arch), nil
	case "windows":
		return fmt.Sprintf("cloudreve_%s_windows_%s.zip", tag, arch), nil
	default:
		return "", fmt.Errorf("unsupported platform for self-update: %s", runtime.GOOS)
	}
}
