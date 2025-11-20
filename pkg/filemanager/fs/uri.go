package fs

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/samber/lo"
)

const (
	Separator = "/"
)

const (
	QuerySearchName           = "name"
	QuerySearchNameOpOr       = "name_op_or"
	QuerySearchUseOr          = "use_or"
	QuerySearchMetadataPrefix = "meta_"
	QuerySearchMetadataExact  = "exact_meta_"
	QuerySearchCaseFolding    = "case_folding"
	QuerySearchType           = "type"
	QuerySearchTypeCategory   = "category"
	QuerySearchSizeGte        = "size_gte"
	QuerySearchSizeLte        = "size_lte"
	QuerySearchCreatedGte     = "created_gte"
	QuerySearchCreatedLte     = "created_lte"
	QuerySearchUpdatedGte     = "updated_gte"
	QuerySearchUpdatedLte     = "updated_lte"
)

type URI struct {
	U *url.URL
}

func NewUriFromString(u string) (*URI, error) {
	raw, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uri: %w", err)
	}

	if raw.Scheme != constants.CloudreveScheme {
		return nil, fmt.Errorf("unknown scheme: %s", raw.Scheme)
	}

	if strings.HasSuffix(raw.Path, Separator) {
		raw.Path = strings.TrimSuffix(raw.Path, Separator)
	}

	return &URI{U: raw}, nil
}

func NewUriFromStrings(u ...string) ([]*URI, error) {
	res := make([]*URI, 0, len(u))
	for _, uri := range u {
		fsUri, err := NewUriFromString(uri)
		if err != nil {
			return nil, err
		}

		res = append(res, fsUri)
	}

	return res, nil
}

func (u *URI) UnmarshalBinary(text []byte) error {
	raw, err := url.Parse(string(text))
	if err != nil {
		return fmt.Errorf("failed to parse uri: %w", err)
	}

	u.U = raw
	return nil
}

func (u *URI) MarshalBinary() ([]byte, error) {
	return u.U.MarshalBinary()
}

func (u *URI) MarshalJSON() ([]byte, error) {
	r := map[string]string{
		"uri": u.String(),
	}
	return json.Marshal(r)
}

func (u *URI) UnmarshalJSON(text []byte) error {
	r := make(map[string]string)
	err := json.Unmarshal(text, &r)
	if err != nil {
		return err
	}

	u.U, err = url.Parse(r["uri"])
	if err != nil {
		return err
	}

	return nil
}

func (u *URI) String() string {
	return u.U.String()
}

func (u *URI) Name() string {
	return path.Base(u.Path())
}

func (u *URI) Dir() string {
	return path.Dir(u.Path())
}

func (u *URI) Elements() []string {
	res := strings.Split(u.PathTrimmed(), Separator)
	if len(res) == 1 && res[0] == "" {
		return nil
	}

	return res
}

func (u *URI) ID(defaultUid string) string {
	if u.U.User == nil {
		if u.FileSystem() != constants.FileSystemShare {
			return defaultUid
		}
		return ""
	}

	return u.U.User.Username()
}

func (u *URI) Path() string {
	p := u.U.Path
	if !strings.HasPrefix(u.U.Path, Separator) {
		p = Separator + u.U.Path
	}

	return path.Clean(p)
}

func (u *URI) PathTrimmed() string {
	return strings.TrimPrefix(u.Path(), Separator)
}

func (u *URI) Password() string {
	if u.U.User == nil {
		return ""
	}

	pwd, _ := u.U.User.Password()
	return pwd
}

func (u *URI) Join(elem ...string) *URI {
	newUrl, _ := url.Parse(u.U.String())
	return &URI{U: newUrl.JoinPath(lo.Map(elem, func(s string, i int) string {
		return PathEscape(s)
	})...)}
}

// Join path with raw string
func (u *URI) JoinRaw(elem string) *URI {
	return u.Join(strings.Split(strings.TrimPrefix(elem, Separator), Separator)...)
}

func (u *URI) DirUri() *URI {
	newUrl, _ := url.Parse(u.U.String())
	newUrl.Path = path.Dir(newUrl.Path)

	return &URI{U: newUrl}
}

func (u *URI) Root() *URI {
	newUrl, _ := url.Parse(u.U.String())
	newUrl.Path = Separator
	newUrl.RawQuery = ""

	return &URI{U: newUrl}
}

func (u *URI) SetQuery(q string) *URI {
	newUrl, _ := url.Parse(u.U.String())
	newUrl.RawQuery = q
	return &URI{U: newUrl}
}

func (u *URI) IsSame(p *URI, uid string) bool {
	return p.FileSystem() == u.FileSystem() && p.ID(uid) == u.ID(uid) && u.Path() == p.Path()
}

// Rebased returns a new URI with the path rebased to the given base URI. It is
// commnly used in WebDAV address translation with shared folder symlink.
func (u *URI) Rebase(target, base *URI) *URI {
	targetPath := target.Path()
	basePath := base.Path()
	rebasedPath := strings.TrimPrefix(targetPath, basePath)

	newUrl, _ := url.Parse(u.U.String())
	newUrl.Path = path.Join(newUrl.Path, rebasedPath)
	return &URI{U: newUrl}
}

func (u *URI) FileSystem() constants.FileSystemType {
	return constants.FileSystemType(strings.ToLower(u.U.Host))
}

// SearchParameters returns the search parameters from the URI. If no search parameters are present, nil is returned.
func (u *URI) SearchParameters() *inventory.SearchFileParameters {
	q := u.U.Query()
	res := &inventory.SearchFileParameters{
		Metadata: make([]inventory.MetadataFilter, 0),
	}
	withSearch := false

	if names, ok := q[QuerySearchName]; ok {
		withSearch = len(names) > 0
		res.Name = names
	}

	if _, ok := q[QuerySearchNameOpOr]; ok {
		res.NameOperatorOr = true
	}

	if _, ok := q[QuerySearchUseOr]; ok {
		res.NameOperatorOr = true
	}

	if _, ok := q[QuerySearchCaseFolding]; ok {
		res.CaseFolding = true
	}

	if v, ok := q[QuerySearchTypeCategory]; ok {
		res.Category = v[0]
		withSearch = withSearch || len(res.Category) > 0
	}

	if t, ok := q[QuerySearchType]; ok {
		fileType := types.FileTypeFromString(t[0])
		res.Type = &fileType
		withSearch = true
	}

	for k, v := range q {
		if strings.HasPrefix(k, QuerySearchMetadataPrefix) {
			res.Metadata = append(res.Metadata, inventory.MetadataFilter{
				Key:   strings.TrimPrefix(k, QuerySearchMetadataPrefix),
				Value: v[0],
				Exact: false,
			})
			withSearch = true
		} else if strings.HasPrefix(k, QuerySearchMetadataExact) {
			res.Metadata = append(res.Metadata, inventory.MetadataFilter{
				Key:   strings.TrimPrefix(k, QuerySearchMetadataExact),
				Value: v[0],
				Exact: true,
			})
			withSearch = true
		}
	}

	if v, ok := q[QuerySearchSizeGte]; ok {
		limit, err := strconv.ParseInt(v[0], 10, 64)
		if err == nil {
			res.SizeGte = limit
			withSearch = true
		}
	}

	if v, ok := q[QuerySearchSizeLte]; ok {
		limit, err := strconv.ParseInt(v[0], 10, 64)
		if err == nil {
			res.SizeLte = limit
			withSearch = true
		}
	}

	if v, ok := q[QuerySearchCreatedGte]; ok {
		limit, err := strconv.ParseInt(v[0], 10, 64)
		if err == nil {
			limit := time.Unix(limit, 0)
			res.CreatedAtGte = &limit
			withSearch = true
		}
	}

	if v, ok := q[QuerySearchCreatedLte]; ok {
		limit, err := strconv.ParseInt(v[0], 10, 64)
		if err == nil {
			limit := time.Unix(limit, 0)
			res.CreatedAtLte = &limit
			withSearch = true
		}
	}

	if v, ok := q[QuerySearchUpdatedGte]; ok {
		limit, err := strconv.ParseInt(v[0], 10, 64)
		if err == nil {
			limit := time.Unix(limit, 0)
			res.UpdatedAtGte = &limit
			withSearch = true
		}
	}

	if v, ok := q[QuerySearchUpdatedLte]; ok {
		limit, err := strconv.ParseInt(v[0], 10, 64)
		if err == nil {
			limit := time.Unix(limit, 0)
			res.UpdatedAtLte = &limit
			withSearch = true
		}
	}

	if withSearch {
		return res
	}

	return nil
}

// EqualOrIsDescendantOf returns true if the URI is equal to the given URI or if it is a descendant of the given URI.
func (u *URI) EqualOrIsDescendantOf(p *URI, uid string) bool {
	prefix := p.Path()
	if prefix[len(prefix)-1] != Separator[0] {
		prefix += Separator
	}

	return p.FileSystem() == u.FileSystem() && p.ID(uid) == u.ID(uid) &&
		(strings.HasPrefix(u.Path(), prefix) || u.Path() == p.Path())
}

func SearchCategoryFromString(s string) setting.SearchCategory {
	switch s {
	case "image":
		return setting.CategoryImage
	case "video":
		return setting.CategoryVideo
	case "audio":
		return setting.CategoryAudio
	case "document":
		return setting.CategoryDocument
	default:
		return setting.CategoryUnknown
	}
}

func NewShareUri(id, password string) string {
	if password != "" {
		return fmt.Sprintf("%s://%s:%s@%s", constants.CloudreveScheme, id, password, constants.FileSystemShare)
	}
	return fmt.Sprintf("%s://%s@%s", constants.CloudreveScheme, id, constants.FileSystemShare)
}

func NewMyUri(id string) string {
	if id == "" {
		return fmt.Sprintf("%s://%s", constants.CloudreveScheme, constants.FileSystemMy)
	}
	return fmt.Sprintf("%s://%s@%s", constants.CloudreveScheme, id, constants.FileSystemMy)
}

// PathEscape is same as url.PathEscape, with modifications to incoporate with JS encodeURIComponent:
// encodeURI() escapes all characters except:
//
//	A–Z a–z 0–9 - _ . ! ~ * ' ( )
func PathEscape(s string) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			hexCount++
		}
	}

	if hexCount == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	if hexCount == 0 {
		copy(t, s)
		for i := 0; i < len(s); i++ {
			if s[i] == ' ' {
				t[i] = '+'
			}
		}
		return string(t)
	}

	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case shouldEscape(c):
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}

const upperhex = "0123456789ABCDEF"

// Return true if the specified character should be escaped when
// appearing in a URL string, according to RFC 3986.
//
// Please be informed that for now shouldEscape does not check all
// reserved characters correctly. See golang.org/issue/5684.
func shouldEscape(c byte) bool {
	// §2.3 Unreserved characters (alphanum)
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}

	switch c {
	case '-', '_', '.', '~', '!', '*', '\'', '(', ')': // §2.3 Unreserved characters (mark)
		return false
	}

	// Everything else must be escaped.
	return true
}
