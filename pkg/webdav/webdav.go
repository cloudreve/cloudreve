// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package webdav provides a WebDAV server implementation.
package webdav // import "golang.org/x/net/webdav"

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/lock"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"math"
)

const (
	davPrefix = "/dav"
)

func stripPrefix(p string, u *ent.User) (string, *fs.URI, int, error) {
	// Bearer-authenticated users carry no dav account; mount them at their
	// "my" root. Basic-auth users keep their configured mount point (#3548).
	baseUri := fs.NewMyUri("")
	if len(u.Edges.DavAccounts) > 0 {
		baseUri = u.Edges.DavAccounts[0].URI
	}
	base, err := fs.NewUriFromString(baseUri)
	if err != nil {
		return "", nil, http.StatusInternalServerError, err
	}

	prefix := davPrefix
	if r := strings.TrimPrefix(p, prefix); len(r) < len(p) {
		r = strings.TrimPrefix(r, fs.Separator)
		return r, base.JoinRaw(util.RemoveSlash(r)), http.StatusOK, nil
	}
	return "", nil, http.StatusNotFound, errPrefixMismatch
}

func ServeHTTP(c *gin.Context) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	fm := manager.NewFileManager(dep, u)
	defer fm.Recycle()

	status, err := http.StatusBadRequest, errUnsupportedMethod

	switch c.Request.Method {
	case "OPTIONS":
		status, err = handleOptions(c, u, fm)
	case "GET", "HEAD", "POST":
		status, err = handleGetHeadPost(c, u, fm)
	case "DELETE":
		status, err = handleDelete(c, u, fm)
	case "PUT":
		status, err = handlePut(c, u, fm)
	case "MKCOL":
		status, err = handleMkcol(c, u, fm)
	case "COPY", "MOVE":
		status, err = handleCopyMove(c, u, fm)
	case "LOCK":
		status, err = handleLock(c, u, fm)
	case "UNLOCK":
		status, err = handleUnlock(c, u, fm)
	case "PROPFIND":
		status, err = handlePropfind(c, u, fm)
	case "PROPPATCH":
		status, err = handleProppatch(c, u, fm)
	}
	if status != 0 {
		c.Writer.WriteHeader(status)
		if status != http.StatusNoContent {
			c.Writer.Write([]byte(StatusText(status)))
		}
	}

	if err != nil {
		dep.Logger().Debug("WebDAV request failed with error: %s", err)
	}
}

func confirmLock(c *gin.Context, fm manager.FileManager, user *ent.User, srcAnc, dstAnc fs.File, src, dst *fs.URI) (func(), fs.LockSession, int, error) {
	hdr := c.Request.Header.Get("If")
	if hdr == "" {
		// An empty If header means that the client hasn't previously created locks.
		// Even if this client doesn't care about locks, we still need to check that
		// the resources aren't locked by another client, so we create temporary
		// locks that would conflict with another client's locks. These temporary
		// locks are unlocked at the end of the HTTP request.
		srcToken, dstToken := "", ""
		ap := fs.LockApp(fs.ApplicationDAV)
		var (
			ctx context.Context = c
			ls  fs.LockSession
			err error
		)
		if src != nil {
			ls, err = fm.Lock(ctx, -1, user, true, ap, src, "")
			if err != nil {
				return nil, nil, purposeStatusCodeFromError(err), err
			}
			srcToken = ls.LastToken()
			ctx = fs.LockSessionToContext(ctx, ls)
		}

		if dst != nil {
			ls, err = fm.Lock(ctx, -1, user, true, ap, dst, "")
			if err != nil {
				if src != nil {
					_ = fm.Unlock(ctx, srcToken)
				}
				return nil, nil, purposeStatusCodeFromError(err), err
			}
			dstToken = ls.LastToken()
			ctx = fs.LockSessionToContext(ctx, ls)
		}

		return func() {
			if dstToken != "" {
				_ = fm.Unlock(ctx, dstToken)
			}
			if srcToken != "" {
				_ = fm.Unlock(ctx, srcToken)
			}
		}, ls, 0, nil
	}

	ih, ok := parseIfHeader(hdr)
	if !ok {
		return nil, nil, http.StatusBadRequest, errInvalidIfHeader
	}
	// ih is a disjunction (OR) of ifLists, so any ifList will do.
	for _, l := range ih.lists {
		var (
			releaseSrc = func() {}
			releaseDst = func() {}
			ls         fs.LockSession
			err        error
		)
		if src != nil {
			releaseSrc, ls, err = fm.ConfirmLock(c, srcAnc, src, lo.Map(l.conditions, func(c Condition, index int) string {
				return c.Token
			})...)
			if errors.Is(err, lock.ErrConfirmationFailed) {
				continue
			}
			if err != nil {
				return nil, nil, purposeStatusCodeFromError(err), err
			}
		}

		if dst != nil {
			releaseDst, ls, err = fm.ConfirmLock(c, dstAnc, dst, lo.Map(l.conditions, func(c Condition, index int) string {
				return c.Token
			})...)
			if errors.Is(err, lock.ErrConfirmationFailed) {
				continue
			}
			if err != nil {
				return nil, nil, purposeStatusCodeFromError(err), err
			}
		}

		return func() {
			releaseDst()
			releaseSrc()
		}, ls, 0, nil
	}
	// Section 10.4.1 says that "If this header is evaluated and all state lists
	// fail, then the request must fail with a 412 (Precondition Failed) status."
	// We follow the spec even though the cond_put_corrupt_token test case from
	// the litmus test warns on seeing a 412 instead of a 423 (Locked).
	return nil, nil, http.StatusPreconditionFailed, ErrLocked
}

// davWriteForbidden reports whether the user is confined to read-only WebDAV
// access, either by a group-level restriction or a read-only dav account.
// Read-only access allows GET/HEAD/OPTIONS/PROPFIND and rejects every
// state-changing method.
func davWriteForbidden(user *ent.User) bool {
	if user == nil {
		return false
	}
	if len(user.Edges.DavAccounts) > 0 &&
		user.Edges.DavAccounts[0].Options.Enabled(int(types.DavAccountReadOnly)) {
		return true
	}
	if inventory.EffectiveGroup(user) != nil &&
		inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionWebDAVReadOnly)) {
		return true
	}
	return false
}

func handleMkcol(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	if davWriteForbidden(user) {
		return http.StatusForbidden, nil
	}
	_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	ancestor, uri, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil && !ent.IsNotFound(err) {
		return purposeStatusCodeFromError(err), err
	}

	release, ls, status, err := confirmLock(c, fm, user, ancestor, nil, uri, nil)
	if err != nil {
		return status, err
	}
	defer release()
	ctx := fs.LockSessionToContext(c, ls)

	if c.Request.ContentLength > 0 {
		return http.StatusUnsupportedMediaType, nil
	}

	_, err = fm.Create(ctx, uri, types.FileTypeFolder, dbfs.WithNoChainedCreation(), dbfs.WithErrorOnConflict())
	if err != nil {
		code := purposeStatusCodeFromError(err)
		if code == http.StatusNotFound {
			// When the MKCOL operation creates a new collection resource, all ancestors MUST already exist,
			// or the method MUST fail with a 409 (Conflict) status code.
			return http.StatusConflict, err
		}
		return code, err
	}

	return http.StatusCreated, nil
}

func handlePut(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	if davWriteForbidden(user) {
		return http.StatusForbidden, nil
	}
	_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	ancestor, uri, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil && !ent.IsNotFound(err) {
		return purposeStatusCodeFromError(err), err
	}

	if len(user.Edges.DavAccounts) > 0 &&
		user.Edges.DavAccounts[0].Options.Enabled(int(types.DavAccountDisableSysFiles)) {
		if strings.HasPrefix(reqPath.Name(), ".") {
			return http.StatusMethodNotAllowed, nil
		}
	}

	release, ls, status, err := confirmLock(c, fm, user, ancestor, nil, uri, nil)
	if err != nil {
		return status, err
	}
	defer release()

	ctx := fs.LockSessionToContext(c, ls)
	// TODO(rost): Support the If-Match, If-None-Match headers? See bradfitz'
	// comments in http.checkEtag.

	rc, fileSize, err := request.SniffContentLength(c.Request)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// A PUT with no length information at all (e.g. chunked transfer encoding)
	// cannot be sized — previously this silently created an empty file.
	if fileSize == 0 && c.Request.ContentLength < 0 &&
		c.Request.Header.Get("X-Expected-Entity-Length") == "" {
		return http.StatusLengthRequired, nil
	}

	// Ranged PUTs ("Content-Range: bytes start-end/total") are used by some
	// clients (e.g. Mountain Duck) to upload large files in pieces. A partial
	// range goes through the chunked-assembly path; a full-range PUT falls
	// through to the regular overwrite path.
	contentRange, err := parseContentRange(c.Request.Header.Get("Content-Range"))
	if err != nil {
		return http.StatusBadRequest, err
	}

	m := manager.NewFileManager(dependency.FromContext(ctx), user)
	defer m.Recycle()

	if contentRange != nil {
		if contentRange.start != 0 || contentRange.end+1 != contentRange.total {
			return handleRangedPut(ctx, c, user, m, fm, uri, rc, fileSize, contentRange)
		}
		if contentRange.total != fileSize {
			return http.StatusBadRequest, errInvalidContentRange
		}
	}

	fileData := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri: uri,
			//MimeType: c.Request.Header.Get("Content-Type"),
			Size: fileSize,
		},
		File: rc,
		Mode: fs.ModeOverwrite,
	}

	// Update file
	res, err := m.Update(ctx, fileData)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	etag, err := findETag(ctx, fm, res)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	c.Writer.Header().Set("ETag", etag)
	return http.StatusCreated, nil
}

var errInvalidContentRange = errors.New("invalid Content-Range")

// contentRange describes a "bytes start-end/total" request range, with end
// inclusive per RFC 7233.
type contentRange struct {
	start, end, total int64
}

// parseContentRange parses a Content-Range header of the form
// "bytes start-end/total". It returns (nil, nil) when the header is absent.
func parseContentRange(h string) (*contentRange, error) {
	if h == "" {
		return nil, nil
	}

	h = strings.TrimSpace(h)
	if !strings.HasPrefix(h, "bytes ") {
		return nil, errInvalidContentRange
	}

	rangePart, totalPart, ok := strings.Cut(h[len("bytes "):], "/")
	if !ok || totalPart == "*" || totalPart == "" {
		// An unknown total cannot be turned into a sized upload session.
		return nil, errInvalidContentRange
	}

	startPart, endPart, ok := strings.Cut(rangePart, "-")
	if !ok {
		return nil, errInvalidContentRange
	}

	start, err := strconv.ParseInt(strings.TrimSpace(startPart), 10, 64)
	if err != nil {
		return nil, errInvalidContentRange
	}
	end, err := strconv.ParseInt(strings.TrimSpace(endPart), 10, 64)
	if err != nil {
		return nil, errInvalidContentRange
	}
	total, err := strconv.ParseInt(strings.TrimSpace(totalPart), 10, 64)
	if err != nil {
		return nil, errInvalidContentRange
	}

	if start < 0 || end < start || total <= 0 || end >= total {
		return nil, errInvalidContentRange
	}
	return &contentRange{start: start, end: end, total: total}, nil
}

// handleRangedPut assembles a multi-request ranged PUT into a single upload
// session. Byte-range coverage is tracked on the session in KV; the upload is
// completed once [0,total) has been received. Only local storage policies are
// supported, as remote drivers cannot honor arbitrary write offsets.
func handleRangedPut(ctx context.Context, c *gin.Context, user *ent.User, m manager.FileManager, fm manager.FileManager, uri *fs.URI, rc request.LimitReaderCloser, fileSize int64, cr *contentRange) (status int, err error) {
	if fileSize != cr.end-cr.start+1 {
		return http.StatusBadRequest, errInvalidContentRange
	}

	dep := dependency.FromContext(c)
	kv := dep.KV()

	// One in-flight ranged upload per user+path, keyed deterministically so
	// that subsequent chunk requests resume the same upload session.
	sessionKey := fmt.Sprintf("dav-put-%d-%x", user.ID, sha1.Sum([]byte(uri.String())))

	var session *fs.UploadSession
	if raw, ok := kv.Get(manager.UploadSessionCachePrefix + sessionKey); ok {
		s, ok := raw.(fs.UploadSession)
		if !ok || s.Props == nil {
			kv.Delete(manager.UploadSessionCachePrefix, sessionKey)
		} else if s.Props.Size == cr.total {
			session = &s
		} else {
			// A different upload to the same path — fail the stale session.
			m.OnUploadFailed(ctx, &s)
			session = nil
		}
	}

	if session == nil {
		ttl := dep.SettingProvider().UploadSessionTTL(ctx)
		if _, err := m.CreateUploadSession(ctx, &fs.UploadRequest{
			Props: &fs.UploadProps{
				Uri:             uri,
				Size:            cr.total,
				UploadSessionID: sessionKey,
				ExpireAt:        time.Now().Add(ttl),
			},
			Mode: fs.ModeOverwrite,
		}); err != nil {
			return purposeStatusCodeFromError(err), err
		}

		raw, ok := kv.Get(manager.UploadSessionCachePrefix + sessionKey)
		if !ok {
			return http.StatusInternalServerError, errors.New("upload session not persisted")
		}
		s := raw.(fs.UploadSession)
		session = &s

		// Only the local driver honors arbitrary write offsets; for other
		// policies a ranged PUT cannot be assembled safely.
		if session.Policy == nil || session.Policy.Type != types.PolicyTypeLocal {
			m.OnUploadFailed(ctx, session)
			return http.StatusNotImplemented, errors.New("ranged PUT not supported by this storage policy")
		}
	}

	chunkReq := &fs.UploadRequest{
		File:   rc,
		Offset: cr.start,
		Props:  session.Props.Copy(),
		Mode:   fs.ModeOverwrite,
	}
	if err := m.Upload(ctx, chunkReq, session.Policy, session); err != nil {
		return purposeStatusCodeFromError(err), err
	}

	if lrc, ok := chunkReq.File.(request.LimitReaderCloser); ok && lrc.Count() != fileSize {
		return http.StatusInternalServerError, fmt.Errorf("uploaded data(%d) does not match purposed size(%d)", lrc.Count(), fileSize)
	}

	allReceived, err := m.MarkRangeUploaded(ctx, session, cr.start, fileSize)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !allReceived {
		// If the session record vanished mid-upload, coverage can never
		// complete — fail so the client retries rather than leaving a stuck
		// placeholder.
		if _, ok := kv.Get(manager.UploadSessionCachePrefix + sessionKey); !ok {
			return http.StatusConflict, errors.New("upload session expired")
		}
		return http.StatusCreated, nil
	}

	res, err := m.CompleteUpload(ctx, session)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	etag, err := findETag(ctx, fm, res)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	c.Writer.Header().Set("ETag", etag)
	return http.StatusCreated, nil
}

func handleOptions(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	allow := []string{"OPTIONS", "LOCK", "PUT", "MKCOL"}

	if user != nil {
		_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
		if err != nil {
			return status, err
		}
		if target, _, err := fm.SharedAddressTranslation(c, reqPath); err == nil {
			allow = allow[:1]
			read, update, del, create := true, true, true, true
			if target.OwnerID() != user.ID || davWriteForbidden(user) {
				update = false
				del = false
				create = false
			}
			if del {
				allow = append(allow, "DELETE", "MOVE")
			}
			if read {
				if !davWriteForbidden(user) {
					allow = append(allow, "COPY")
				}
				allow = append(allow, "PROPFIND")
				if target.Type() == types.FileTypeFile {
					allow = append(allow, "GET", "HEAD", "POST")
				}
			}
			if update || create {
				allow = append(allow, "LOCK", "UNLOCK")
			}
			if update {
				allow = append(allow, "PROPPATCH")
				if target.Type() == types.FileTypeFile {
					allow = append(allow, "PUT")
				}
			}
		} else {
			logging.FromContext(c).Debug("Handle options failed to get target: %s", err)
		}
	}

	c.Writer.Header().Set("Allow", strings.Join(allow, ", "))
	// http://www.webdav.org/specs/rfc4918.html#dav.compliance.classes
	c.Writer.Header().Set("DAV", "1, 2")
	// http://msdn.microsoft.com/en-au/library/cc250217.aspx
	c.Writer.Header().Set("MS-Author-Via", "DAV")
	return 0, nil
}

func handleGetHeadPost(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	target, _, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	if target.Type() != types.FileTypeFile {
		return http.StatusMethodNotAllowed, nil
	}

	es, err := fm.GetEntitySource(c, target.PrimaryEntityID())
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	defer es.Close()

	es.Apply(entitysource.WithSpeedLimit(int64(inventory.EffectiveGroup(user).SpeedLimit)))
	if es.ShouldInternalProxy() ||
		(len(user.Edges.DavAccounts) > 0 &&
			user.Edges.DavAccounts[0].Options.Enabled(int(types.DavAccountProxy)) &&
			inventory.EffectiveGroup(user).Permissions.Enabled(int(types.GroupPermissionWebDAVProxy))) {
		es.Serve(c.Writer, c.Request)
	} else {
		settings := dependency.FromContext(c).SettingProvider()
		expire := time.Now().Add(settings.EntityUrlValidDuration(c))
		src, err := es.Url(c, entitysource.WithExpire(&expire))
		if err != nil {
			return purposeStatusCodeFromError(err), err
		}
		c.Redirect(http.StatusFound, src.Url)
	}

	return 0, nil
}

func handleUnlock(c *gin.Context, user *ent.User, fm manager.FileManager) (retStatus int, retErr error) {
	// http://www.webdav.org/specs/rfc4918.html#HEADER_Lock-Token says that the
	// Lock-Token value is a Coded-URL. We strip its angle brackets.
	t := c.Request.Header.Get("Lock-Token")
	if len(t) < 2 || t[0] != '<' || t[len(t)-1] != '>' {
		return http.StatusBadRequest, errInvalidLockToken
	}
	t = t[1 : len(t)-1]
	err := fm.Unlock(c, t)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	return http.StatusNoContent, err
}

func handleLock(c *gin.Context, user *ent.User, fm manager.FileManager) (retStatus int, retErr error) {
	if davWriteForbidden(user) {
		return http.StatusForbidden, nil
	}
	duration, err := parseTimeout(c.Request.Header.Get("Timeout"))
	if err != nil {
		return http.StatusBadRequest, err
	}
	li, status, err := readLockInfo(c.Request.Body)
	if err != nil {
		return status, err
	}

	href, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	token, ld, created := "", lock.LockDetails{}, false
	if li == (lockInfo{}) {
		// An empty lockInfo means to refresh the lock.
		ih, ok := parseIfHeader(c.Request.Header.Get("If"))
		if !ok {
			return http.StatusBadRequest, errInvalidIfHeader
		}
		if len(ih.lists) == 1 && len(ih.lists[0].conditions) == 1 {
			token = ih.lists[0].conditions[0].Token
		}
		if token == "" {
			return http.StatusBadRequest, errInvalidLockToken
		}
		ld, err = fm.Refresh(c, duration, token)
		if err != nil {
			if errors.Is(err, lock.ErrNoSuchLock) {
				return http.StatusPreconditionFailed, err
			}
			return http.StatusInternalServerError, err
		}
		ld.Root = href
	} else {
		// Section 9.10.3 says that "If no Depth header is submitted on a LOCK request,
		// then the request MUST act as if a "Depth:infinity" had been submitted."
		depth := infiniteDepth
		if hdr := c.Request.Header.Get("Depth"); hdr != "" {
			depth = parseDepth(hdr)
			if depth != 0 && depth != infiniteDepth {
				// Section 9.10.3 says that "Values other than 0 or infinity must not be
				// used with the Depth header on a LOCK method".
				return http.StatusBadRequest, errInvalidDepth
			}
		}

		ancestor, uri, err := fm.SharedAddressTranslation(c, reqPath)
		if err != nil && !ent.IsNotFound(err) {
			return purposeStatusCodeFromError(err), err
		}

		ld = lock.LockDetails{
			Root:      href,
			Duration:  duration,
			Owner:     lock.Owner{Application: lock.Application{InnerXML: li.Owner.InnerXML}},
			ZeroDepth: depth == 0,
		}
		app := lock.Application{
			Type:     string(fs.ApplicationDAV),
			InnerXML: li.Owner.InnerXML,
		}
		ls, err := fm.Lock(c, duration, user, depth == 0, app, uri, "")
		if err != nil {
			if errors.Is(err, lock.ErrLocked) {
				return StatusLocked, err
			}
			return http.StatusInternalServerError, err
		}
		token = ls.LastToken()
		ctx := fs.LockSessionToContext(c, ls)
		defer func() {
			if retErr != nil {
				_ = fm.Unlock(c, token)
			}
		}()

		// Create the resource if it didn't previously exist.
		hasher := dependency.FromContext(c).HashIDEncoder()
		if !ancestor.Uri(false).IsSame(uri, hashid.EncodeUserID(hasher, user.ID)) {
			if _, err = fm.Create(ctx, uri, types.FileTypeFile); err != nil {
				return purposeStatusCodeFromError(err), err
			}

			created = true
		}

		// http://www.webdav.org/specs/rfc4918.html#HEADER_Lock-Token says that the
		// Lock-Token value is a Coded-URL. We add angle brackets.
		c.Writer.Header().Set("Lock-Token", "<"+token+">")
	}

	c.Writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
	if created {
		// This is "w.WriteHeader(http.StatusCreated)" and not "return
		// http.StatusCreated, nil" because we write our own (XML) response to w
		// and Handler.ServeHTTP would otherwise write "Created".
		c.Writer.WriteHeader(http.StatusCreated)
	}
	writeLockInfo(c.Writer, token, ld)
	return 0, nil
}

func handlePropfind(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	href, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	_, targetPath, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	depth := infiniteDepth
	if hdr := c.Request.Header.Get("Depth"); hdr != "" {
		depth = parseDepth(hdr)
		if depth == invalidDepth {
			return http.StatusBadRequest, errInvalidDepth
		}
	}
	pf, status, err := readPropfind(c.Request.Body)
	if err != nil {
		return status, err
	}

	mw := multistatusWriter{w: c.Writer}
	walkFn := func(f fs.File, level int) error {
		var pstats []Propstat
		if pf.Propname != nil {
			pnames, err := propnames(c, f, fm)
			if err != nil {
				return err
			}
			pstat := Propstat{Status: http.StatusOK}
			for _, xmlname := range pnames {
				pstat.Props = append(pstat.Props, Property{XMLName: xmlname})
			}
			pstats = append(pstats, pstat)
		} else if pf.Allprop != nil {
			pstats, err = allprop(c, f, fm, pf.Prop)
		} else {
			pstats, err = props(c, f, fm, pf.Prop)
		}
		if err != nil {
			return err
		}

		p := path.Join(davPrefix, href)
		elements := f.Uri(false).Elements()
		for i := 0; i < level; i++ {
			p = path.Join(p, elements[len(elements)-level+i])
		}
		if f.Type() == types.FileTypeFolder {
			p = util.FillSlash(p)
		}

		return mw.write(makePropstatResponse(p, pstats))
	}

	if err := fm.Walk(c, targetPath, depth, walkFn, dbfs.WithFilePublicMetadata()); err != nil {
		return purposeStatusCodeFromError(err), err
	}

	closeErr := mw.close()
	if closeErr != nil {
		return http.StatusInternalServerError, closeErr
	}
	return 0, nil
}

func handleDelete(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	if davWriteForbidden(user) {
		return http.StatusForbidden, nil
	}
	_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	ancestor, uri, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	release, ls, status, err := confirmLock(c, fm, user, ancestor, nil, uri, nil)
	if err != nil {
		return status, err
	}
	defer release()
	ctx := fs.LockSessionToContext(c, ls)

	// TODO: return MultiStatus where appropriate.

	if err := fm.Delete(ctx, []*fs.URI{uri}); err != nil {
		return purposeStatusCodeFromError(err), err
	}

	return http.StatusNoContent, nil
}

func handleCopyMove(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	if davWriteForbidden(user) {
		return http.StatusForbidden, nil
	}
	hdr := c.Request.Header.Get("Destination")
	if hdr == "" {
		return http.StatusBadRequest, errInvalidDestination
	}
	u, err := url.Parse(hdr)
	if err != nil {
		return http.StatusBadRequest, errInvalidDestination
	}
	if u.Host != "" && u.Host != c.Request.Host {
		return http.StatusBadGateway, errInvalidDestination
	}

	_, src, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	srcTarget, srcUri, err := fm.SharedAddressTranslation(c, src)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	_, dst, status, err := stripPrefix(u.Path, user)
	if err != nil {
		return status, err
	}

	dstTarget, dstUri, err := fm.SharedAddressTranslation(c, dst)
	if err != nil && !ent.IsNotFound(err) {
		return purposeStatusCodeFromError(err), err
	}
	dstExists := err == nil

	_, dstFolderUri, err := fm.SharedAddressTranslation(c, dst.DirUri())
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	overwrite, err := parseOverwrite(c.Request.Header.Get("Overwrite"))
	if err != nil {
		return http.StatusBadRequest, err
	}

	hasher := dependency.FromContext(c).HashIDEncoder()
	if srcUri.IsSame(dstUri, hashid.EncodeUserID(hasher, user.ID)) {
		return http.StatusForbidden, errDestinationEqualsSource
	}

	if c.Request.Method == "COPY" {
		// Section 7.5.1 says that a COPY only needs to lock the destination,
		// not both destination and source. Strictly speaking, this is racy,
		// even though a COPY doesn't modify the source, if a concurrent
		// operation modifies the source. However, the litmus test explicitly
		// checks that COPYing a locked-by-another source is OK.
		release, ls, status, err := confirmLock(c, fm, user, dstTarget, nil, dstUri, nil)
		if err != nil {
			return status, err
		}
		defer release()
		ctx := fs.LockSessionToContext(c, ls)

		// Section 9.8.3 says that "The COPY method on a collection without a Depth
		// header must act as if a Depth header with value "infinity" was included".
		depth := infiniteDepth
		if hdr := c.Request.Header.Get("Depth"); hdr != "" {
			depth = parseDepth(hdr)
			if depth != 0 && depth != infiniteDepth {
				// Section 9.8.3 says that "A client may submit a Depth header on a
				// COPY on a collection with a value of "0" or "infinity"."
				return http.StatusBadRequest, errInvalidDepth
			}
		}

		return performCopyMove(ctx, fm, srcUri, dstUri, dstFolderUri, true, overwrite, dstExists, hashid.EncodeUserID(hasher, user.ID))
	}

	release, ls, status, err := confirmLock(c, fm, user, srcTarget, dstTarget, srcUri, dstUri)
	if err != nil {
		return status, err
	}
	defer release()
	ctx := fs.LockSessionToContext(c, ls)

	// Section 9.9.2 says that "The MOVE method on a collection must act as if
	// a "Depth: infinity" header was used on it. A client must not submit a
	// Depth header on a MOVE on a collection with any value but "infinity"."
	if hdr := c.Request.Header.Get("Depth"); hdr != "" {
		if parseDepth(hdr) != infiniteDepth {
			return http.StatusBadRequest, errInvalidDepth
		}
	}
	return performCopyMove(ctx, fm, srcUri, dstUri, dstFolderUri, false, overwrite, dstExists, hashid.EncodeUserID(hasher, user.ID))
}

type copyMoveOperations interface {
	Delete(ctx context.Context, path []*fs.URI, opts ...fs.Option) error
	MoveOrCopy(ctx context.Context, src []*fs.URI, dst *fs.URI, isCopy bool) error
	Rename(ctx context.Context, path *fs.URI, newName string) (fs.File, error)
	SharedAddressTranslation(ctx context.Context, path *fs.URI, opts ...fs.Option) (fs.File, *fs.URI, error)
}

func parseOverwrite(value string) (bool, error) {
	switch value {
	case "", "T":
		return true, nil
	case "F":
		return false, nil
	default:
		return false, errInvalidOverwrite
	}
}

func performCopyMove(
	ctx context.Context,
	fm copyMoveOperations,
	srcUri, dstUri, dstFolderUri *fs.URI,
	isCopy, overwrite, dstExists bool,
	uid string,
) (int, error) {
	if srcUri.Name() != dstUri.Name() {
		// A renamed move/copy lands at dstFolder/srcName before the rename;
		// a different resource already occupying that path would hit the
		// UNIQUE(parent, name) constraint mid-operation.
		_, intermediateUri, err := fm.SharedAddressTranslation(ctx, dstFolderUri.Join(srcUri.Name()))
		if err == nil && !intermediateUri.IsSame(srcUri, uid) {
			return http.StatusPreconditionFailed, errDestinationExists
		} else if err != nil && !ent.IsNotFound(err) {
			return purposeStatusCodeFromError(err), err
		}
	}

	if dstExists {
		if !overwrite {
			return http.StatusPreconditionFailed, errDestinationExists
		}
		if err := fm.Delete(ctx, []*fs.URI{dstUri}); err != nil {
			return purposeStatusCodeFromError(err), err
		}
	}

	if err := fm.MoveOrCopy(ctx, []*fs.URI{srcUri}, dstFolderUri, isCopy); err != nil {
		return purposeStatusCodeFromError(err), err
	}

	if dstUri.Name() != srcUri.Name() {
		if _, err := fm.Rename(ctx, dstFolderUri.Join(srcUri.Name()), dstUri.Name()); err != nil {
			return purposeStatusCodeFromError(err), err
		}
	}

	if dstExists {
		return http.StatusNoContent, nil
	}
	return http.StatusCreated, nil
}

func handleProppatch(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
	if davWriteForbidden(user) {
		return http.StatusForbidden, nil
	}
	_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	ancestor, uri, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil {
		return purposeStatusCodeFromError(err), err
	}

	release, ls, status, err := confirmLock(c, fm, user, ancestor, nil, uri, nil)
	if err != nil {
		return status, err
	}
	defer release()
	ctx := fs.LockSessionToContext(c, ls)

	patches, status, err := readProppatch(c.Request.Body)
	if err != nil {
		return status, err
	}
	pstats, err := patch(ctx, ancestor, fm, patches)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	mw := multistatusWriter{w: c.Writer}
	writeErr := mw.write(makePropstatResponse(c.Request.URL.Path, pstats))
	closeErr := mw.close()
	if writeErr != nil {
		return http.StatusInternalServerError, writeErr
	}
	if closeErr != nil {
		return http.StatusInternalServerError, closeErr
	}
	return 0, nil
}

func purposeStatusCodeFromError(err error) int {
	if ent.IsNotFound(err) {
		return http.StatusNotFound
	}

	if errors.Is(err, lock.ErrNoSuchLock) {
		return http.StatusConflict
	}

	var ae *serializer.AggregateError
	if errors.As(err, &ae) && len(ae.Raw()) > 0 {
		for _, e := range ae.Raw() {
			return purposeStatusCodeFromError(e)
		}
	}

	var appErr serializer.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case serializer.CodeNotFound, serializer.CodeParentNotExist, serializer.CodeEntityNotExist:
			return http.StatusNotFound
		case serializer.CodeNoPermissionErr:
			return http.StatusForbidden
		case serializer.CodeLockConflict:
			return http.StatusLocked
		case serializer.CodeObjectExist:
			return http.StatusMethodNotAllowed
		}
	}

	return http.StatusInternalServerError
}

func makePropstatResponse(href string, pstats []Propstat) *response {
	resp := response{
		Href:     []string{(&url.URL{Path: href}).EscapedPath()},
		Propstat: make([]propstat, 0, len(pstats)),
	}
	for _, p := range pstats {
		var xmlErr *xmlError
		if p.XMLError != "" {
			xmlErr = &xmlError{InnerXML: []byte(p.XMLError)}
		}
		resp.Propstat = append(resp.Propstat, propstat{
			Status:              fmt.Sprintf("HTTP/1.1 %d %s", p.Status, StatusText(p.Status)),
			Prop:                p.Props,
			ResponseDescription: p.ResponseDescription,
			Error:               xmlErr,
		})
	}
	return &resp
}

const (
	infiniteDepth = math.MaxInt
	invalidDepth  = -2
)

// parseDepth maps the strings "0", "1" and "infinity" to 0, 1 and
// infiniteDepth. Parsing any other string returns invalidDepth.
//
// Different WebDAV methods have further constraints on valid depths:
//   - PROPFIND has no further restrictions, as per section 9.1.
//   - COPY accepts only "0" or "infinity", as per section 9.8.3.
//   - MOVE accepts only "infinity", as per section 9.9.2.
//   - LOCK accepts only "0" or "infinity", as per section 9.10.3.
//
// These constraints are enforced by the handleXxx methods.
func parseDepth(s string) int {
	switch s {
	case "0":
		return 0
	case "1":
		return 1
	case "infinity":
		return infiniteDepth
	}
	return invalidDepth
}

// http://www.webdav.org/specs/rfc4918.html#status.code.extensions.to.http11
const (
	StatusMulti               = 207
	StatusUnprocessableEntity = 422
	StatusLocked              = 423
	StatusFailedDependency    = 424
	StatusInsufficientStorage = 507
)

func StatusText(code int) string {
	switch code {
	case StatusMulti:
		return "Multi-Status"
	case StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case StatusLocked:
		return "Locked"
	case StatusFailedDependency:
		return "Failed Dependency"
	case StatusInsufficientStorage:
		return "Insufficient Storage"
	}
	return http.StatusText(code)
}

var (
	errDestinationEqualsSource = errors.New("webdav: destination equals source")
	errDestinationExists       = errors.New("webdav: destination exists")
	errDirectoryNotEmpty       = errors.New("webdav: directory not empty")
	errInvalidDepth            = errors.New("webdav: invalid depth")
	errInvalidDestination      = errors.New("webdav: invalid destination")
	errInvalidIfHeader         = errors.New("webdav: invalid If header")
	errInvalidLockInfo         = errors.New("webdav: invalid lock info")
	errInvalidLockToken        = errors.New("webdav: invalid lock token")
	errInvalidOverwrite        = errors.New("webdav: invalid overwrite")
	errInvalidPropfind         = errors.New("webdav: invalid propfind")
	errInvalidProppatch        = errors.New("webdav: invalid proppatch")
	errInvalidResponse         = errors.New("webdav: invalid response")
	errInvalidTimeout          = errors.New("webdav: invalid timeout")
	errNoFileSystem            = errors.New("webdav: no file system")
	errNoLockSystem            = errors.New("webdav: no lock system")
	errNotADirectory           = errors.New("webdav: not a directory")
	errPrefixMismatch          = errors.New("webdav: prefix mismatch")
	errRecursionTooDeep        = errors.New("webdav: recursion too deep")
	errUnsupportedLockInfo     = errors.New("webdav: unsupported lock info")
	errUnsupportedMethod       = errors.New("webdav: unsupported method")
)
