// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdav

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//// 实现 webdav.DeadPropsHolder 接口，不能在models.file里面定义
//func (file *FileDeadProps) DeadProps() (map[xml.Name]Property, error) {
//	return map[xml.Name]Property{
//		xml.Name{Space: "http://owncloud.org/ns", Local: "checksums"}: {
//			XMLName: xml.Name{
//				Space: "http://owncloud.org/ns", Local: "checksums",
//			},
//			InnerXML: []byte("<checksum>" + file.MetadataSerialized[model.ChecksumMetadataKey] + "</checksum>"),
//		},
//	}, nil
//}
//
//func (file *FileDeadProps) Patch(proppatches []Proppatch) ([]Propstat, error) {
//	var (
//		stat Propstat
//		err  error
//	)
//	stat.Status = http.StatusOK
//	for _, patch := range proppatches {
//		for _, prop := range patch.Props {
//			stat.Props = append(stat.Props, Property{XMLName: prop.XMLName})
//			if prop.XMLName.Space == "DAV:" && prop.XMLName.Local == "lastmodified" {
//				var modtimeUnix int64
//				modtimeUnix, err = strconv.ParseInt(string(prop.InnerXML), 10, 64)
//				if err == nil {
//					err = model.DB.Model(file).UpdateColumn("updated_at", time.Unix(modtimeUnix, 0)).Error
//				}
//			}
//		}
//	}
//	return []Propstat{stat}, err
//}
//
//type FolderDeadProps struct {
//	*model.Folder
//}
//
//func (folder *FolderDeadProps) DeadProps() (map[xml.Name]Property, error) {
//	return nil, nil
//}
//
//func (folder *FolderDeadProps) Patch(proppatches []Proppatch) ([]Propstat, error) {
//	var (
//		stat Propstat
//		err  error
//	)
//	stat.Status = http.StatusOK
//	for _, patch := range proppatches {
//		for _, prop := range patch.Props {
//			stat.Props = append(stat.Props, Property{XMLName: prop.XMLName})
//			if prop.XMLName.Space == "DAV:" && prop.XMLName.Local == "lastmodified" {
//				var modtimeUnix int64
//				modtimeUnix, err = strconv.ParseInt(string(prop.InnerXML), 10, 64)
//				if err == nil {
//					err = model.DB.Model(folder).UpdateColumn("updated_at", time.Unix(modtimeUnix, 0)).Error
//				}
//			}
//		}
//	}
//	return []Propstat{stat}, err
//}

const (
	DeadPropsMetadataPrefix = "dav:"
	SpaceNameSeparator      = "|"
)

type (
	// DeadPropsStore implements  DeadPropsHolder interface with metadata based store.
	metadataDeadProps struct {
		f  fs.File
		fm manager.FileManager
	}

	DeadPropsStore struct {
		Lang     string `json:"l,omitempty"`
		InnerXML []byte `json:"i,omitempty"`
	}
)

func (m *metadataDeadProps) DeadProps() (map[xml.Name]Property, error) {
	meta := m.f.Metadata()
	res := make(map[xml.Name]Property)
	for k, v := range meta {
		if !strings.HasPrefix(k, DeadPropsMetadataPrefix) {
			continue
		}

		spaceLocal := strings.SplitN(strings.TrimPrefix(k, DeadPropsMetadataPrefix), SpaceNameSeparator, 2)
		name := xml.Name{Space: spaceLocal[0], Local: spaceLocal[1]}
		propsStore := &DeadPropsStore{}
		if err := json.Unmarshal([]byte(v), propsStore); err != nil {
			return nil, err
		}

		res[name] = Property{
			XMLName:  name,
			InnerXML: propsStore.InnerXML,
			Lang:     propsStore.Lang,
		}
	}

	return res, nil
}

func (m *metadataDeadProps) Patch(ctx context.Context, proppatches []Proppatch) ([]Propstat, error) {
	metadataArgs := make([]fs.MetadataPatch, 0, len(proppatches))
	pstat := Propstat{Status: http.StatusOK}
	for _, patch := range proppatches {
		translateFn := func(p Property) (*fs.MetadataPatch, error) {
			val, err := json.Marshal(&DeadPropsStore{
				Lang:     p.Lang,
				InnerXML: p.InnerXML,
			})
			if err != nil {
				return nil, err
			}
			return &fs.MetadataPatch{
				Key:   DeadPropsMetadataPrefix + p.XMLName.Space + SpaceNameSeparator + p.XMLName.Local,
				Value: string(val),
			}, nil
		}
		if patch.Remove {
			translateFn = func(p Property) (*fs.MetadataPatch, error) {
				return &fs.MetadataPatch{
					Key:    DeadPropsMetadataPrefix + p.XMLName.Space + SpaceNameSeparator + p.XMLName.Local,
					Remove: true,
				}, nil
			}
		}
		for _, prop := range patch.Props {
			pstat.Props = append(pstat.Props, Property{XMLName: prop.XMLName})
			patch, err := translateFn(prop)
			if err != nil {
				return nil, err
			}
			metadataArgs = append(metadataArgs, *patch)
		}
	}

	if err := m.fm.PatchMedata(ctx, []*fs.URI{m.f.Uri(false)}, metadataArgs...); err != nil {
		return nil, err
	}

	return []Propstat{pstat}, nil
}

type FileInfo interface {
	GetSize() uint64
	GetName() string
	ModTime() time.Time
	IsDir() bool
	GetPosition() string
}

// Proppatch describes a property update instruction as defined in RFC 4918.
// See http://www.webdav.org/specs/rfc4918.html#METHOD_PROPPATCH
type Proppatch struct {
	// Remove specifies whether this patch removes properties. If it does not
	// remove them, it sets them.
	Remove bool
	// Props contains the properties to be set or removed.
	Props []Property
}

// Propstat describes a XML propstat element as defined in RFC 4918.
// See http://www.webdav.org/specs/rfc4918.html#ELEMENT_propstat
type Propstat struct {
	// Props contains the properties for which Status applies.
	Props []Property

	// Status defines the HTTP status code of the properties in Prop.
	// Allowed values include, but are not limited to the WebDAV status
	// code extensions for HTTP/1.1.
	// http://www.webdav.org/specs/rfc4918.html#status.code.extensions.to.http11
	Status int

	// XMLError contains the XML representation of the optional error element.
	// XML content within this field must not rely on any predefined
	// namespace declarations or prefixes. If empty, the XML error element
	// is omitted.
	XMLError string

	// ResponseDescription contains the contents of the optional
	// responsedescription field. If empty, the XML element is omitted.
	ResponseDescription string
}

// makePropstats returns a slice containing those of x and y whose Props slice
// is non-empty. If both are empty, it returns a slice containing an otherwise
// zero Propstat whose HTTP status code is 200 OK.
func makePropstats(x, y Propstat) []Propstat {
	pstats := make([]Propstat, 0, 2)
	if len(x.Props) != 0 {
		pstats = append(pstats, x)
	}
	if len(y.Props) != 0 {
		pstats = append(pstats, y)
	}
	if len(pstats) == 0 {
		pstats = append(pstats, Propstat{
			Status: http.StatusOK,
		})
	}
	return pstats
}

// DeadPropsHolder holds the dead properties of a resource.
//
// Dead properties are those properties that are explicitly defined. In
// comparison, live properties, such as DAV:getcontentlength, are implicitly
// defined by the underlying resource, and cannot be explicitly overridden or
// removed. See the Terminology section of
// http://www.webdav.org/specs/rfc4918.html#rfc.section.3
//
// There is a whitelist of the names of live properties. This package handles
// all live properties, and will only pass non-whitelisted names to the Patch
// method of DeadPropsHolder implementations.
type DeadPropsHolder interface {
	// DeadProps returns a copy of the dead properties held.
	DeadProps() (map[xml.Name]Property, error)

	// Patch patches the dead properties held.
	//
	// Patching is atomic; either all or no patches succeed. It returns (nil,
	// non-nil) if an internal server error occurred, otherwise the Propstats
	// collectively contain one Property for each proposed patch Property. If
	// all patches succeed, Patch returns a slice of length one and a Propstat
	// element with a 200 OK HTTP status code. If none succeed, for reasons
	// other than an internal server error, no Propstat has status 200 OK.
	//
	// For more details on when various HTTP status codes apply, see
	// http://www.webdav.org/specs/rfc4918.html#PROPPATCH-status
	Patch(context.Context, []Proppatch) ([]Propstat, error)
}

// liveProps contains all supported, protected DAV: properties.
var liveProps = map[xml.Name]struct {
	// findFn implements the propfind function of this property. If nil,
	// it indicates a hidden property.
	findFn func(context.Context, manager.FileManager, fs.File) (string, error)
	// dir is true if the property applies to directories.
	dir bool
}{
	{Space: "DAV:", Local: "resourcetype"}: {
		findFn: findResourceType,
		dir:    true,
	},
	{Space: "DAV:", Local: "displayname"}: {
		findFn: findDisplayName,
		dir:    true,
	},
	{Space: "DAV:", Local: "getcontentlength"}: {
		findFn: findContentLength,
		dir:    false,
	},
	{Space: "DAV:", Local: "getlastmodified"}: {
		findFn: findLastModified,
		// http://webdav.org/specs/rfc4918.html#PROPERTY_getlastmodified
		// suggests that getlastmodified should only apply to GETable
		// resources, and this package does not support GET on directories.
		//
		// Nonetheless, some WebDAV clients expect child directories to be
		// sortable by getlastmodified date, so this value is true, not false.
		// See golang.org/issue/15334.
		dir: true,
	},
	{Space: "DAV:", Local: "creationdate"}: {
		findFn: findCreationDate,
		dir:    true,
	},
	{Space: "DAV:", Local: "getcontentlanguage"}: {
		findFn: nil,
		dir:    false,
	},
	{Space: "DAV:", Local: "getcontenttype"}: {
		findFn: findContentType,
		dir:    false,
	},
	{Space: "DAV:", Local: "getetag"}: {
		findFn: findETag,
		// findETag implements ETag as the concatenated hex values of a file's
		// modification time and size. This is not a reliable synchronization
		// mechanism for directories, so we do not advertise getetag for DAV
		// collections.
		dir: false,
	},

	// TODO: The lockdiscovery property requires LockSystem to list the
	// active locks on a resource.
	{Space: "DAV:", Local: "lockdiscovery"}: {},
	{Space: "DAV:", Local: "supportedlock"}: {
		findFn: findSupportedLock,
		dir:    true,
	},
	{Space: "DAV:", Local: "quota-used-bytes"}: {
		findFn: findQuotaUsedBytes,
		dir:    true,
	},
	{Space: "DAV:", Local: "quota-available-bytes"}: {
		findFn: findQuotaAvailableBytes,
		dir:    true,
	},
}

// TODO(nigeltao) merge props and allprop?

// Props returns the status of the properties named pnames for resource name.
//
// Each Propstat has a unique status and each property name will only be part
// of one Propstat element.
func props(c *gin.Context, file fs.File, fm manager.FileManager, pnames []xml.Name) ([]Propstat, error) {
	isDir := file.Type() == types.FileTypeFolder
	dph := &metadataDeadProps{
		f:  file,
		fm: fm,
	}

	var (
		deadProps map[xml.Name]Property
		err       error
	)
	deadProps, err = dph.DeadProps()
	if err != nil {
		return nil, err
	}

	pstatOK := Propstat{Status: http.StatusOK}
	pstatNotFound := Propstat{Status: http.StatusNotFound}
	for _, pn := range pnames {
		// If this file has dead properties, check if they contain pn.
		if dp, ok := deadProps[pn]; ok {
			pstatOK.Props = append(pstatOK.Props, dp)
			continue
		}
		// Otherwise, it must either be a live property or we don't know it.
		if prop := liveProps[pn]; prop.findFn != nil && (prop.dir || !isDir) {
			innerXML, err := prop.findFn(c, fm, file)
			if err != nil {
				if errors.Is(err, ErrNotImplemented) {
					pstatNotFound.Props = append(pstatNotFound.Props, Property{
						XMLName: pn,
					})
					continue
				}
				return nil, err
			}
			pstatOK.Props = append(pstatOK.Props, Property{
				XMLName:  pn,
				InnerXML: []byte(innerXML),
			})
		} else {
			pstatNotFound.Props = append(pstatNotFound.Props, Property{
				XMLName: pn,
			})
		}
	}
	return makePropstats(pstatOK, pstatNotFound), nil
}

// Propnames returns the property names defined for resource name.
func propnames(c *gin.Context, file fs.File, fm manager.FileManager) ([]xml.Name, error) {
	var deadProps map[xml.Name]Property
	dph := &metadataDeadProps{
		f:  file,
		fm: fm,
	}
	deadProps, err := dph.DeadProps()
	if err != nil {
		return nil, err
	}

	isDir := file.Type() == types.FileTypeFolder
	pnames := make([]xml.Name, 0, len(liveProps)+len(deadProps))
	for pn, prop := range liveProps {
		if prop.findFn != nil && (prop.dir || !isDir) {
			pnames = append(pnames, pn)
		}
	}
	for pn := range deadProps {
		pnames = append(pnames, pn)
	}
	return pnames, nil
}

// Allprop returns the properties defined for resource name and the properties
// named in include.
//
// Note that RFC 4918 defines 'allprop' to return the DAV: properties defined
// within the RFC plus dead properties. Other live properties should only be
// returned if they are named in 'include'.
//
// See http://www.webdav.org/specs/rfc4918.html#METHOD_PROPFIND
func allprop(c *gin.Context, file fs.File, fm manager.FileManager, include []xml.Name) ([]Propstat, error) {
	pnames, err := propnames(c, file, fm)
	if err != nil {
		return nil, err
	}
	// Add names from include if they are not already covered in pnames.
	nameset := make(map[xml.Name]bool)
	for _, pn := range pnames {
		nameset[pn] = true
	}
	for _, pn := range include {
		if !nameset[pn] {
			pnames = append(pnames, pn)
		}
	}
	return props(c, file, fm, pnames)
}

// Patch patches the properties of resource name. The return values are
// constrained in the same manner as DeadPropsHolder.Patch.
func patch(c context.Context, file fs.File, fm manager.FileManager, patches []Proppatch) ([]Propstat, error) {
	conflict := false
loop:
	for _, patch := range patches {
		for _, p := range patch.Props {
			if _, ok := liveProps[p.XMLName]; ok {
				conflict = true
				break loop
			}
		}
	}
	if conflict {
		pstatForbidden := Propstat{
			Status:   http.StatusForbidden,
			XMLError: `<D:cannot-modify-protected-property xmlns:D="DAV:"/>`,
		}
		pstatFailedDep := Propstat{
			Status: StatusFailedDependency,
		}
		for _, patch := range patches {
			for _, p := range patch.Props {
				if _, ok := liveProps[p.XMLName]; ok {
					pstatForbidden.Props = append(pstatForbidden.Props, Property{XMLName: p.XMLName})
				} else {
					pstatFailedDep.Props = append(pstatFailedDep.Props, Property{XMLName: p.XMLName})
				}
			}
		}
		return makePropstats(pstatForbidden, pstatFailedDep), nil
	}

	// very unlikely to be false
	dph := &metadataDeadProps{
		f:  file,
		fm: fm,
	}

	ret, err := dph.Patch(c, patches)
	if err != nil {
		return nil, err
	}
	// http://www.webdav.org/specs/rfc4918.html#ELEMENT_propstat says that
	// "The contents of the prop XML element must only list the names of
	// properties to which the result in the status element applies."
	for _, pstat := range ret {
		for i, p := range pstat.Props {
			pstat.Props[i] = Property{XMLName: p.XMLName}
		}
	}
	return ret, nil
}

func escapeXML(s string) string {
	for i := 0; i < len(s); i++ {
		// As an optimization, if s contains only ASCII letters, digits or a
		// few special characters, the escaped value is s itself and we don't
		// need to allocate a buffer and convert between string and []byte.
		switch c := s[i]; {
		case c == ' ' || c == '_' ||
			('+' <= c && c <= '9') || // Digits as well as + , - . and /
			('A' <= c && c <= 'Z') ||
			('a' <= c && c <= 'z'):
			continue
		}
		// Otherwise, go through the full escaping process.
		var buf bytes.Buffer
		xml.EscapeText(&buf, []byte(s))
		return buf.String()
	}
	return s
}

// ErrNotImplemented should be returned by optional interfaces if they
// want the original implementation to be used.
var ErrNotImplemented = errors.New("not implemented")

func findResourceType(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	if file.Type() == types.FileTypeFolder {
		return `<D:collection xmlns:D="DAV:"/>`, nil
	}
	return "", nil
}

func findDisplayName(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	return escapeXML(file.DisplayName()), nil
}

func findContentLength(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	return strconv.FormatInt(file.Size(), 10), nil
}

func findLastModified(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	return file.UpdatedAt().UTC().Format(http.TimeFormat), nil
}

func findCreationDate(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	return file.CreatedAt().UTC().Format(http.TimeFormat), nil
}

func findContentType(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	d := dependency.FromContext(ctx)
	return d.MimeDetector(ctx).TypeByName(file.DisplayName()), nil
}

func findETag(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	hasher := dependency.FromContext(ctx).HashIDEncoder()
	return fmt.Sprintf(`"%s"`, hashid.EncodeEntityID(hasher, file.PrimaryEntityID())), nil
}

func findQuotaUsedBytes(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	requester := inventory.UserFromContext(ctx)
	if file.Owner().ID != requester.ID {
		return "", ErrNotImplemented
	}
	capacity, err := fm.Capacity(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(capacity.Used, 10), nil
}

func findQuotaAvailableBytes(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	requester := inventory.UserFromContext(ctx)
	if file.Owner().ID != requester.ID {
		return "", ErrNotImplemented
	}
	capacity, err := fm.Capacity(ctx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(capacity.Total-capacity.Used, 10), nil
}

func findSupportedLock(ctx context.Context, fm manager.FileManager, file fs.File) (string, error) {
	return `` +
		`<D:lockentry xmlns:D="DAV:">` +
		`<D:lockscope><D:exclusive/></D:lockscope>` +
		`<D:locktype><D:write/></D:locktype>` +
		`</D:lockentry>`, nil
}
