package share

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
)

type LoadStatus int

const (
	LoadOK LoadStatus = iota
	LoadNotFound
	LoadExpired
	LoadError
)

// LoadShareForInfo loads share info for public preview/metadata access.
func LoadShareForInfo(ctx context.Context, shareClient inventory.ShareClient, shareID int, viewer *ent.User, password string) (*ent.Share, bool, LoadStatus, error) {
	ctx = context.WithValue(ctx, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	share, err := shareClient.GetByID(ctx, shareID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, false, LoadNotFound, nil
		}
		return nil, false, LoadError, err
	}

	if err := inventory.IsValidShare(share); err != nil {
		return share, false, LoadExpired, err
	}

	unlocked := isShareUnlocked(share, password, viewer)
	return share, unlocked, LoadOK, nil
}

// BuildRedirectURL builds the long share URL and merges query params safely.
func BuildRedirectURL(id, password, sharePath string, extraQuery url.Values) string {
	sharePath = SanitizeSharePath(sharePath)
	shareLongURL := routes.MasterShareLongUrl(id, password)
	shareLongURLQuery := shareLongURL.Query()

	if sharePath != "" {
		masterPath := shareLongURLQuery.Get("path")
		masterPath += "/" + strings.TrimPrefix(sharePath, "/")
		shareLongURLQuery.Set("path", masterPath)
	}

	for key, vals := range extraQuery {
		if key == "path" {
			continue
		}
		shareLongURLQuery[key] = append(shareLongURLQuery[key], vals...)
	}

	shareLongURL.RawQuery = shareLongURLQuery.Encode()
	return shareLongURL.String()
}

// SanitizeSharePath normalizes a share path for use in URLs.
func SanitizeSharePath(raw string) string {
	if raw == "" {
		return ""
	}

	cleaned := path.Clean("/" + raw)
	return strings.TrimPrefix(cleaned, "/")
}

func isShareUnlocked(share *ent.Share, password string, viewer *ent.User) bool {
	if share.Password == "" {
		return true
	}
	if password == share.Password {
		return true
	}
	if viewer != nil && share.Edges.User != nil && share.Edges.User.ID == viewer.ID {
		return true
	}
	return false
}
