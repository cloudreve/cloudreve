// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package webdav provides a WebDAV server implementation.
package webdav // import "golang.org/x/net/webdav"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
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
	"golang.org/x/tools/container/intsets"
)

const (
	davPrefix = "/dav"
)

func stripPrefix(p string, u *ent.User) (string, *fs.URI, int, error) {
	base, err := fs.NewUriFromString(u.Edges.DavAccounts[0].URI)
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

func handleMkcol(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
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
	_, reqPath, status, err := stripPrefix(c.Request.URL.Path, user)
	if err != nil {
		return status, err
	}

	ancestor, uri, err := fm.SharedAddressTranslation(c, reqPath)
	if err != nil && !ent.IsNotFound(err) {
		return purposeStatusCodeFromError(err), err
	}

	if user.Edges.DavAccounts[0].Options.Enabled(int(types.DavAccountDisableSysFiles)) {
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

	fileData := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri: uri,
			//MimeType: c.Request.Header.Get("Content-Type"),
			Size: fileSize,
		},
		File: rc,
		Mode: fs.ModeOverwrite,
	}

	m := manager.NewFileManager(dependency.FromContext(ctx), user)
	defer m.Recycle()

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
			if target.OwnerID() != user.ID {
				update = false
				del = false
				create = false
			}
			if del {
				allow = append(allow, "DELETE", "MOVE")
			}
			if read {
				allow = append(allow, "COPY", "PROPFIND")
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

	es.Apply(entitysource.WithSpeedLimit(int64(user.Edges.Group.SpeedLimit)))
	if es.ShouldInternalProxy() ||
		(user.Edges.DavAccounts[0].Options.Enabled(int(types.DavAccountProxy)) &&
			user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionWebDAVProxy))) {
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

	_, dstFolderUri, err := fm.SharedAddressTranslation(c, dst.DirUri())
	if err != nil {
		return purposeStatusCodeFromError(err), err
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

		if err := fm.MoveOrCopy(ctx, []*fs.URI{srcUri}, dstFolderUri, true); err != nil {
			return purposeStatusCodeFromError(err), err
		}
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
	if err := fm.MoveOrCopy(ctx, []*fs.URI{srcUri}, dstFolderUri, false); err != nil {
		return purposeStatusCodeFromError(err), err
	}

	if dstUri.Name() != srcUri.Name() {
		if _, err := fm.Rename(ctx, dstFolderUri.Join(srcUri.Name()), dstUri.Name()); err != nil {
			return purposeStatusCodeFromError(err), err
		}
	}

	return http.StatusNoContent, nil
}

func handleProppatch(c *gin.Context, user *ent.User, fm manager.FileManager) (status int, err error) {
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
	infiniteDepth = intsets.MaxInt
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
	errDirectoryNotEmpty       = errors.New("webdav: directory not empty")
	errInvalidDepth            = errors.New("webdav: invalid depth")
	errInvalidDestination      = errors.New("webdav: invalid destination")
	errInvalidIfHeader         = errors.New("webdav: invalid If header")
	errInvalidLockInfo         = errors.New("webdav: invalid lock info")
	errInvalidLockToken        = errors.New("webdav: invalid lock token")
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
