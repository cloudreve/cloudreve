package routes

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
)

const (
	IsDownloadQuery             = "download"
	IsThumbQuery                = "thumb"
	SlaveClearTaskRegistryQuery = "deleteOnComplete"
)

var (
	masterPing         *url.URL
	masterUserActivate *url.URL
	masterUserReset    *url.URL
	masterHome         *url.URL
)

func init() {
	masterPing, _ = url.Parse(constants.APIPrefix + "/site/ping")
	masterUserActivate, _ = url.Parse("/session/activate")
	masterUserReset, _ = url.Parse("/session/reset")
}

func FrontendHomeUrl(base *url.URL, path string) *url.URL {
	route, _ := url.Parse(fmt.Sprintf("/home"))
	q := route.Query()
	q.Set("path", path)
	route.RawQuery = q.Encode()

	return base.ResolveReference(route)
}

func MasterPingUrl(base *url.URL) *url.URL {
	return base.ResolveReference(masterPing)
}

func MasterSlaveCallbackUrl(base *url.URL, driver, id, secret string) *url.URL {
	apiBaseURI, _ := url.Parse(path.Join(constants.APIPrefix+"/callback", driver, id, secret))
	return base.ResolveReference(apiBaseURI)
}

func MasterUserActivateAPIUrl(base *url.URL, uid string) *url.URL {
	route, _ := url.Parse(constants.APIPrefix + "/user/activate/" + uid)
	return base.ResolveReference(route)
}

func MasterUserActivateUrl(base *url.URL) *url.URL {
	return base.ResolveReference(masterUserActivate)
}

func MasterUserResetUrl(base *url.URL) *url.URL {
	return base.ResolveReference(masterUserReset)
}

func MasterShareUrl(base *url.URL, id, password string) *url.URL {
	p := "/s/" + id
	if password != "" {
		p += ("/" + password)
	}
	route, _ := url.Parse(p)
	return base.ResolveReference(route)
}

func MasterDirectLink(base *url.URL, id, name string) *url.URL {
	p := path.Join("/f", id, url.PathEscape(name))
	route, _ := url.Parse(p)
	return base.ResolveReference(route)
}

// MasterShareLongUrl generates a long share URL for redirect.
func MasterShareLongUrl(id, password string) *url.URL {
	base, _ := url.Parse("/home")
	q := base.Query()

	q.Set("path", fs.NewShareUri(id, password))
	base.RawQuery = q.Encode()
	return base
}

func MasterArchiveDownloadUrl(base *url.URL, sessionID string) *url.URL {
	routes, err := url.Parse(path.Join(constants.APIPrefix, "file", "archive", sessionID, "archive.zip"))
	if err != nil {
		return nil
	}

	return base.ResolveReference(routes)
}

func MasterPolicyOAuthCallback(base *url.URL) *url.URL {
	if base.Scheme != "https" {
		base.Scheme = "https"
	}
	routes, err := url.Parse("/admin/policy/oauth")
	if err != nil {
		return nil
	}
	return base.ResolveReference(routes)
}

func MasterGetCredentialUrl(base, key string) *url.URL {
	masterBase, err := url.Parse(base)
	if err != nil {
		return nil
	}

	routes, err := url.Parse(path.Join(constants.APIPrefixSlave, "credential", key))
	if err != nil {
		return nil
	}

	return masterBase.ResolveReference(routes)
}

func MasterStatelessUrl(base, method string) *url.URL {
	masterBase, err := url.Parse(base)
	if err != nil {
		return nil
	}

	routes, err := url.Parse(path.Join(constants.APIPrefixSlave, "statelessUpload", method))
	if err != nil {
		return nil
	}

	return masterBase.ResolveReference(routes)
}

func SlaveUploadUrl(base *url.URL, sessionID string) *url.URL {
	base.Path = path.Join(base.Path, constants.APIPrefixSlave, "/upload", sessionID)
	return base
}

func MasterFileContentUrl(base *url.URL, entityId, name string, download, thumb bool, speed int64) *url.URL {
	name = url.PathEscape(name)

	route, _ := url.Parse(constants.APIPrefix + fmt.Sprintf("/file/content/%s/%d/%s", entityId, speed, name))
	if base != nil {
		route = base.ResolveReference(route)
	}

	values := url.Values{}
	if download {
		values.Set(IsDownloadQuery, "true")
	}

	if thumb {
		values.Set(IsThumbQuery, "true")
	}

	route.RawQuery = values.Encode()
	return route
}

func MasterWopiSrc(base *url.URL, sessionId string) *url.URL {
	route, _ := url.Parse(constants.APIPrefix + "/file/wopi/" + sessionId)
	return base.ResolveReference(route)
}

func SlaveFileContentUrl(base *url.URL, srcPath, name string, download bool, speed int64, nodeId int) *url.URL {
	srcPath = url.PathEscape(base64.URLEncoding.EncodeToString([]byte(srcPath)))
	name = url.PathEscape(name)
	route, _ := url.Parse(constants.APIPrefixSlave + fmt.Sprintf("/file/content/%d/%s/%d/%s", nodeId, srcPath, speed, name))
	base = base.ResolveReference(route)

	values := url.Values{}
	if download {
		values.Set(IsDownloadQuery, "true")
	}

	base.RawQuery = values.Encode()
	return base
}

func SlaveMediaMetaRoute(src, ext, language string) string {
	src = url.PathEscape(base64.URLEncoding.EncodeToString([]byte(src)))
	return fmt.Sprintf("file/meta/%s/%s?language=%s", src, url.PathEscape(ext), language)
}

func SlaveFileListRoute(srcPath string, recursive bool) string {
	base := "file/list"
	query := url.Values{}
	query.Set("recursive", strconv.FormatBool(recursive))
	query.Set("path", srcPath)
	return fmt.Sprintf("%s?%s", base, query.Encode())
}

func SlaveThumbUrl(base *url.URL, srcPath, ext string) *url.URL {
	srcPath = url.PathEscape(base64.URLEncoding.EncodeToString([]byte(srcPath)))
	ext = url.PathEscape(ext)
	route, _ := url.Parse(constants.APIPrefixSlave + fmt.Sprintf("/file/thumb/%s/%s", srcPath, ext))
	base = base.ResolveReference(route)
	return base
}

func SlaveGetTaskRoute(id int, deleteOnComplete bool) string {
	p := constants.APIPrefixSlave + "/task/" + strconv.Itoa(id)
	if deleteOnComplete {
		p += "?" + SlaveClearTaskRegistryQuery + "=true"
	}
	return p
}

func SlavePingRoute(base *url.URL) string {
	route, _ := url.Parse(constants.APIPrefixSlave + "/ping")
	return base.ResolveReference(route).String()
}
