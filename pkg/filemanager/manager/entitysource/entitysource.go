package entitysource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/local"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/encrypt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/mime"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/juju/ratelimit"
)

const (
	shortSeekBytes = 1024
	// The algorithm uses at most sniffLen bytes to make its decision.
	sniffLen         = 512
	defaultUrlExpire = time.Hour * 1
)

var (
	// ErrNoContentLength is returned by Seek when the initial http response did not include a Content-Length header
	ErrNoContentLength = errors.New("Content-Length was not set")

	// errNoOverlap is returned by serveContent's parseRange if first-byte-pos of
	// all of the byte-range-spec values is greater than the content size.
	errNoOverlap = errors.New("invalid range: failed to overlap")
)

type EntitySource interface {
	io.ReadSeekCloser
	io.ReaderAt

	// Url generates a download url for the entity.
	Url(ctx context.Context, opts ...EntitySourceOption) (*EntityUrl, error)
	// Serve serves the entity to the client, with supports on Range header and If- cache control.
	Serve(w http.ResponseWriter, r *http.Request, opts ...EntitySourceOption)
	// Entity returns the entity of the source.
	Entity() fs.Entity
	// IsLocal returns true if the source is in local machine.
	IsLocal() bool
	// LocalPath returns the local path of the source file.
	LocalPath(ctx context.Context) string
	// Apply applies the options to the source.
	Apply(opts ...EntitySourceOption)
	// CloneToLocalSrc clones the source to a local file source.
	CloneToLocalSrc(t types.EntityType, src string) (EntitySource, error)
	// ShouldInternalProxy returns true if the source will/should be proxied by internal proxy.
	ShouldInternalProxy(opts ...EntitySourceOption) bool
}

type EntitySourceOption interface {
	Apply(any)
}

type EntitySourceOptions struct {
	SpeedLimit         int64
	Expire             *time.Time
	IsDownload         bool
	NoInternalProxy    bool
	DisplayName        string
	OneTimeDownloadKey string
	Ctx                context.Context
	IsThumb            bool
	DisableCryptor     bool
}

type EntityUrl struct {
	Url      string
	ExpireAt *time.Time
}

type EntitySourceOptionFunc func(any)

// WithSpeedLimit set speed limit for file source (if supported)
func WithSpeedLimit(limit int64) EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).SpeedLimit = limit
	})
}

// WithExpire set expire time for file source
func WithExpire(expire *time.Time) EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).Expire = expire
	})
}

// WithDownload set file URL as download
func WithDownload(isDownload bool) EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).IsDownload = isDownload
	})
}

// WithNoInternalProxy overwrite policy's internal proxy setting
func WithNoInternalProxy() EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).NoInternalProxy = true
	})
}

// WithDisplayName set display name for file source
func WithDisplayName(name string) EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).DisplayName = name
	})
}

// WithContext set context for file source
func WithContext(ctx context.Context) EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).Ctx = ctx
	})
}

// WithThumb set entity source as thumb. This will result in entity source URL
// generated with thumbnail processing parameters. For sidecar thumb files,
// this option will be ignored.
func WithThumb(isThumb bool) EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).IsThumb = isThumb
	})
}

// WithDisableCryptor disable cryptor for file source, file stream will be
// presented as is.
func WithDisableCryptor() EntitySourceOption {
	return EntitySourceOptionFunc(func(option any) {
		option.(*EntitySourceOptions).DisableCryptor = true
	})
}

func (f EntitySourceOptionFunc) Apply(option any) {
	f(option)
}

type (
	entitySource struct {
		e                fs.Entity
		handler          driver.Handler
		policy           *ent.StoragePolicy
		generalAuth      auth.Auth
		settings         setting.Provider
		hasher           hashid.Encoder
		c                request.Client
		l                logging.Logger
		config           conf.ConfigProvider
		mime             mime.MimeDetector
		encryptorFactory encrypt.CryptorFactory

		rsc io.ReadCloser
		pos int64
		o   *EntitySourceOptions

		// Cache for resetRequest URL and expiry
		cachedUrl    string
		cachedExpiry time.Time
	}
)

// toASCIISafeFilename converts a filename to ASCII-safe version by replacing
// non-ASCII characters and special characters with underscores.
// This is used for the fallback filename parameter in Content-Disposition header.
func toASCIISafeFilename(filename string) string {
	asciiFilename := ""
	for _, r := range filename {
		if r <= 127 && r >= 32 && r != '"' && r != '\\' {
			asciiFilename += string(r)
		} else {
			asciiFilename += "_"
		}
	}
	return asciiFilename
}

// NewEntitySource creates a new EntitySource.
func NewEntitySource(
	e fs.Entity,
	handler driver.Handler,
	policy *ent.StoragePolicy,
	generalAuth auth.Auth,
	settings setting.Provider,
	hasher hashid.Encoder,
	c request.Client,
	l logging.Logger,
	config conf.ConfigProvider,
	mime mime.MimeDetector,
	encryptorFactory encrypt.CryptorFactory,
	opts ...EntitySourceOption,
) EntitySource {
	s := &entitySource{
		e:                e,
		handler:          handler,
		policy:           policy,
		generalAuth:      generalAuth,
		settings:         settings,
		hasher:           hasher,
		c:                c,
		config:           config,
		l:                l,
		mime:             mime,
		encryptorFactory: encryptorFactory,
		o:                &EntitySourceOptions{},
	}
	for _, opt := range opts {
		opt.Apply(s.o)
	}
	return s
}

func (f *entitySource) Apply(opts ...EntitySourceOption) {
	if len(opts) > 0 {
		// Clear cache when options are applied as they might affect URL generation
		f.clearUrlCache()
	}
	for _, opt := range opts {
		opt.Apply(f.o)
	}
}

func (f *entitySource) CloneToLocalSrc(t types.EntityType, src string) (EntitySource, error) {
	e, err := local.NewLocalFileEntity(t, src)
	if err != nil {
		return nil, err
	}

	policy := &ent.StoragePolicy{Type: types.PolicyTypeLocal}
	handler := local.New(policy, f.l, f.config)

	newSrc := NewEntitySource(e, handler, policy, f.generalAuth, f.settings, f.hasher, f.c, f.l, f.config, f.mime, f.encryptorFactory).(*entitySource)
	newSrc.o = f.o
	return newSrc, nil
}

func (f *entitySource) Entity() fs.Entity {
	return f.e
}

func (f *entitySource) IsLocal() bool {
	return f.handler.Capabilities().StaticFeatures.Enabled(int(driver.HandlerCapabilityInboundGet))
}

func (f *entitySource) LocalPath(ctx context.Context) string {
	return f.handler.LocalPath(ctx, f.e.Source())
}

func (f *entitySource) Serve(w http.ResponseWriter, r *http.Request, opts ...EntitySourceOption) {
	if len(opts) > 0 {
		// Clear cache when options are applied as they might affect URL generation
		f.clearUrlCache()
	}
	for _, opt := range opts {
		opt.Apply(f.o)
	}

	if f.IsLocal() {
		// For local files, validate file existence by resetting rsc
		if err := f.resetRequest(); err != nil {
			f.l.Warning("Failed to serve local entity %q: %s", err, f.e.Source())
			http.Error(w, "Entity data does not exist.", http.StatusNotFound)
			return
		}
	}

	etag := "\"" + hashid.EncodeEntityID(f.hasher, f.e.ID()) + "\""
	w.Header().Set("Etag", "\""+hashid.EncodeEntityID(f.hasher, f.e.ID())+"\"")

	if f.o.IsDownload {
		// Properly handle non-ASCII characters in filename according to RFC 6266
		displayName := f.o.DisplayName
		asciiFilename := toASCIISafeFilename(displayName)

		// RFC 6266 compliant filename* encoding
		encodedFilename := url.QueryEscape(displayName)

		if displayName == asciiFilename {
			// Filename contains only ASCII characters, use simple form
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))
		} else {
			// Filename contains non-ASCII characters, use both parameters
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s",
				asciiFilename, encodedFilename))
		}
	}

	done, rangeReq := checkPreconditions(w, r, etag)
	if done {
		return
	}

	if !f.IsLocal() {
		// for non-local file, reverse-proxy the request
		expire := time.Now().Add(defaultUrlExpire)
		u, err := f.Url(driver.WithForcePublicEndpoint(f.o.Ctx, false), WithNoInternalProxy(), WithExpire(&expire))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		target, err := url.Parse(u.Url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		start := time.Now()
		proxy := &httputil.ReverseProxy{
			Director: func(request *http.Request) {
				request.URL.Scheme = target.Scheme
				request.URL.Host = target.Host
				request.URL.Path = target.Path
				request.URL.RawPath = target.RawPath
				request.URL.RawQuery = target.RawQuery
				request.Host = target.Host
				request.Header.Del("Authorization")
			},
			ModifyResponse: func(response *http.Response) error {
				response.Header.Del("ETag")
				response.Header.Del("Content-Disposition")
				response.Header.Del("Cache-Control")

				// If the response is successful, decrypt the body if needed
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					// Parse offset from Content-Range header if present
					offset := parseContentRangeOffset(response.Header.Get("Content-Range"))

					body, err := f.getDecryptedRsc(response.Body, offset)
					if err != nil {
						return fmt.Errorf("failed to get decrypted rsc: %w", err)
					}

					response.Body = body
				}

				logging.Request(f.l,
					false,
					response.StatusCode,
					response.Request.Method,
					request.LocalIP,
					response.Request.URL.String(),
					"",
					start,
				)
				return nil
			},
			ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
				f.l.Error("Reverse proxy error in %q: %s", request.URL.String(), err)
				writer.WriteHeader(http.StatusBadGateway)
				writer.Write([]byte("[Cloudreve] Bad Gateway"))
			},
		}

		r = r.Clone(f.o.Ctx)
		defer func() {
			if err := recover(); err != nil && err != http.ErrAbortHandler {
				panic(err)
			}
		}()
		proxy.ServeHTTP(w, r)
		return
	}

	code := http.StatusOK
	// If Content-Type isn't set, use the file's extension to find it, but
	// if the Content-Type is unset explicitly, do not sniff the type.
	ctypes, haveType := w.Header()["Content-Type"]
	var ctype string
	if !haveType {
		ctype = f.mime.TypeByName(f.o.DisplayName)
		if ctype == "" {
			// read a chunk to decide between utf-8 text and binary
			var buf [sniffLen]byte
			n, _ := io.ReadFull(f, buf[:])
			ctype = http.DetectContentType(buf[:n])
			_, err := f.Seek(0, io.SeekStart) // rewind to output whole file
			if err != nil {
				http.Error(w, "seeker can't seek", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", ctype)
	} else if len(ctypes) > 0 {
		ctype = ctypes[0]
	}

	size := f.e.Size()
	if size < 0 {
		// Should never happen but just to be sure
		http.Error(w, "negative content size computed", http.StatusInternalServerError)
		return
	}

	// handle Content-Range header.
	sendSize := size
	var sendContent io.Reader = f
	ranges, err := parseRange(rangeReq, size)
	switch err {
	case nil:
	case errNoOverlap:
		if size == 0 {
			// Some clients add a Range header to all requests to
			// limit the size of the response. If the file is empty,
			// ignore the range header and respond with a 200 rather
			// than a 416.
			ranges = nil
			break
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		fallthrough
	default:
		http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if sumRangesSize(ranges) > size {
		// The total number of bytes in all the ranges
		// is larger than the size of the file by
		// itself, so this is probably an attack, or a
		// dumb client. Ignore the range request.
		ranges = nil
	}
	switch {
	case len(ranges) == 1:
		// RFC 7233, Section 4.1:
		// "If a single part is being transferred, the server
		// generating the 206 response MUST generate a
		// Content-Range header field, describing what range
		// of the selected representation is enclosed, and a
		// payload consisting of the range.
		// ...
		// A server MUST NOT generate a multipart response to
		// a request for a single range, since a client that
		// does not request multiple parts might not support
		// multipart responses."
		ra := ranges[0]
		if _, err := f.Seek(ra.start, io.SeekStart); err != nil {
			http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
			return
		}
		sendSize = ra.length
		code = http.StatusPartialContent
		w.Header().Set("Content-Range", ra.contentRange(size))
	case len(ranges) > 1:
		sendSize = rangesMIMESize(ranges, ctype, size)
		code = http.StatusPartialContent

		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)
		w.Header().Set("Content-Type", "multipart/byteranges; boundary="+mw.Boundary())
		sendContent = pr
		defer pr.Close() // cause writing goroutine to fail and exit if CopyN doesn't finish.
		go func() {
			for _, ra := range ranges {
				part, err := mw.CreatePart(ra.mimeHeader(ctype, size))
				if err != nil {
					pw.CloseWithError(err)
					return
				}
				if _, err := f.Seek(ra.start, io.SeekStart); err != nil {
					pw.CloseWithError(err)
					return
				}
				if _, err := io.CopyN(part, f, ra.length); err != nil {
					pw.CloseWithError(err)
					return
				}
			}
			mw.Close()
			pw.Close()
		}()
	}

	w.Header().Set("Accept-Ranges", "bytes")
	if w.Header().Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))
	}

	w.WriteHeader(code)

	if r.Method != "HEAD" {
		io.CopyN(w, sendContent, sendSize)
	}
}

func (f *entitySource) Read(p []byte) (n int, err error) {
	if f.rsc == nil {
		err = f.resetRequest()
	}
	if f.rsc != nil {
		n, err = f.rsc.Read(p)
		f.pos += int64(n)
	}
	return
}

func (f *entitySource) ReadAt(p []byte, off int64) (n int, err error) {
	if f.rsc == nil {
		err = f.resetRequest()
		if err != nil {
			return 0, err
		}
	}
	if readAt, ok := f.rsc.(io.ReaderAt); ok {
		return readAt.ReadAt(p, off)
	}

	// For non-local sources, use HTTP range request to read at specific offset
	rsc, err := f.getRsc(off)
	if err != nil {
		return 0, err
	}
	return io.ReadFull(rsc, p)
}

func (f *entitySource) Seek(offset int64, whence int) (int64, error) {
	var err error
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += f.pos
	case io.SeekEnd:
		offset = f.e.Size() + offset
	}
	if f.rsc != nil {
		// Try to read, which is cheaper than doing a request
		if f.pos < offset && offset-f.pos <= shortSeekBytes {
			_, err := io.CopyN(io.Discard, f, offset-f.pos)
			if err != nil {
				return 0, err
			}
		}

		if f.pos != offset {
			err = f.rsc.Close()
			f.rsc = nil
		}
	}
	f.pos = offset
	return f.pos, err
}

func (f *entitySource) Close() error {
	if f.rsc != nil {
		return f.rsc.Close()
	}
	return nil
}

// clearUrlCache clears the cached URL and expiry
func (f *entitySource) clearUrlCache() {
	f.cachedUrl = ""
	f.cachedExpiry = time.Time{}
}

func (f *entitySource) ShouldInternalProxy(opts ...EntitySourceOption) bool {
	for _, opt := range opts {
		opt.Apply(f.o)
	}
	handlerCapability := f.handler.Capabilities()
	return f.e.ID() == 0 || handlerCapability.StaticFeatures.Enabled(int(driver.HandlerCapabilityProxyRequired)) ||
		(f.policy.Settings.InternalProxy || f.e.Encrypted()) && !f.o.NoInternalProxy
}

func (f *entitySource) Url(ctx context.Context, opts ...EntitySourceOption) (*EntityUrl, error) {
	if len(opts) > 0 {
		// Clear cache when options are applied as they might affect URL generation
		f.clearUrlCache()
	}
	for _, opt := range opts {
		opt.Apply(f.o)
	}

	var (
		srcUrl    *url.URL
		err       error
		srcUrlStr string
	)

	expire := f.o.Expire
	displayName := f.o.DisplayName
	if displayName == "" {
		displayName = path.Base(util.FormSlash(f.e.Source()))
	}

	// Use internal proxy URL if:
	// 1. Internal proxy is required by driver's definition
	// 2. Internal proxy is enabled in Policy setting and not disabled by option
	// 3. It's an empty entity.
	// 4. The entity is encrypted and internal proxy not disabled by option
	handlerCapability := f.handler.Capabilities()
	if f.ShouldInternalProxy() {
		siteUrl := f.settings.SiteURL(ctx)
		base := routes.MasterFileContentUrl(
			siteUrl,
			hashid.EncodeEntityID(f.hasher, f.e.ID()),
			displayName,
			f.o.IsDownload,
			f.o.IsThumb,
			f.o.SpeedLimit,
		)

		srcUrl, err = auth.SignURI(ctx, f.generalAuth, base.String(), expire)
		if err != nil {
			return nil, fmt.Errorf("failed to sign internal proxy URL: %w", err)
		}

		if f.IsLocal() {
			// For local file, we need to apply proxy if needed
			srcUrl, err = driver.ApplyProxyIfNeeded(f.policy, srcUrl)
			if err != nil {
				return nil, fmt.Errorf("failed to apply proxy: %w", err)
			}
		}
	} else {
		expire = capExpireTime(expire, handlerCapability.MinSourceExpire, handlerCapability.MaxSourceExpire)
		if f.o.IsThumb {
			srcUrlStr, err = f.handler.Thumb(ctx, expire, util.Ext(f.o.DisplayName), f.e)
		} else {
			srcUrlStr, err = f.handler.Source(ctx, f.e, &driver.GetSourceArgs{
				Expire:      expire,
				IsDownload:  f.o.IsDownload,
				Speed:       f.o.SpeedLimit,
				DisplayName: displayName,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get source URL: %w", err)
		}

		srcUrl, err = url.Parse(srcUrlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse origin URL: %w", err)
		}

		srcUrl, err = driver.ApplyProxyIfNeeded(f.policy, srcUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to apply proxy: %w", err)
		}
	}

	return &EntityUrl{
		Url:      srcUrl.String(),
		ExpireAt: expire,
	}, nil
}

func (f *entitySource) resetRequest() error {
	// For inbound files, we can use the handler to open the file directly
	if f.IsLocal() && f.rsc != nil {
		return nil
	}

	rsc, err := f.getRsc(f.pos)
	if err != nil {
		return fmt.Errorf("failed to get rsc: %w", err)
	}
	f.rsc = rsc
	return nil
}

func (f *entitySource) getRsc(pos int64) (io.ReadCloser, error) {
	// For inbound files, we can use the handler to open the file directly
	var rsc io.ReadCloser
	if f.IsLocal() {
		file, err := f.handler.Open(f.o.Ctx, f.e.Source())
		if err != nil {
			return nil, fmt.Errorf("failed to open inbound file: %w", err)
		}

		if pos > 0 {
			_, err = file.Seek(pos, io.SeekStart)
			if err != nil {
				return nil, fmt.Errorf("failed to seek inbound file: %w", err)
			}
		}

		if f.o.SpeedLimit > 0 {
			bucket := ratelimit.NewBucketWithRate(float64(f.o.SpeedLimit), f.o.SpeedLimit)
			rsc = lrs{file, ratelimit.Reader(file, bucket)}
		} else {
			rsc = file
		}
	} else {
		var urlStr string
		now := time.Now()

		// Check if we have a valid cached URL and expiry
		if f.cachedUrl != "" && now.Before(f.cachedExpiry.Add(-time.Minute)) {
			// Use cached URL if it's still valid (with 1 minute buffer before expiry)
			urlStr = f.cachedUrl
		} else {
			// Generate new URL and cache it
			expire := now.Add(defaultUrlExpire)
			u, err := f.Url(driver.WithForcePublicEndpoint(f.o.Ctx, false), WithNoInternalProxy(), WithExpire(&expire))
			if err != nil {
				return nil, fmt.Errorf("failed to generate download url: %w", err)
			}

			// Cache the URL and expiry
			f.cachedUrl = u.Url
			f.cachedExpiry = expire
			urlStr = u.Url
		}

		h := http.Header{}
		h.Set("Range", fmt.Sprintf("bytes=%d-", pos))
		resp := f.c.Request(http.MethodGet, urlStr, nil,
			request.WithContext(f.o.Ctx),
			request.WithLogger(f.l),
			request.WithHeader(h),
		).CheckHTTPResponse(http.StatusOK, http.StatusPartialContent)
		if resp.Err != nil {
			return nil, fmt.Errorf("failed to request download url: %w", resp.Err)
		}

		rsc = resp.Response.Body
	}

	var err error
	rsc, err = f.getDecryptedRsc(rsc, pos)
	if err != nil {
		return nil, fmt.Errorf("failed to get decrypted rsc: %w", err)
	}

	return rsc, nil
}

func (f *entitySource) getDecryptedRsc(rsc io.ReadCloser, pos int64) (io.ReadCloser, error) {
	props := f.e.Props()
	if props != nil && props.EncryptMetadata != nil && !f.o.DisableCryptor {
		cryptor, err := f.encryptorFactory(props.EncryptMetadata.Algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to create decryptor: %w", err)
		}
		err = cryptor.LoadMetadata(f.o.Ctx, props.EncryptMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to load metadata: %w", err)
		}

		if err := cryptor.SetSource(rsc, nil, f.e.Size(), pos); err != nil {
			return nil, fmt.Errorf("failed to set source: %w", err)
		}

		return cryptor, nil
	}

	return rsc, nil
}

// capExpireTime make sure expire time is not too long or too short (if min or max is set)
func capExpireTime(expire *time.Time, min, max time.Duration) *time.Time {
	timeNow := time.Now()
	if expire == nil {
		return nil
	}

	cappedExpires := *expire
	// Make sure expire time is not too long or too short
	if min > 0 && expire.Before(timeNow.Add(min)) {
		cappedExpires = timeNow.Add(min)
	} else if max > 0 && expire.After(timeNow.Add(max)) {
		cappedExpires = timeNow.Add(max)
	}

	return &cappedExpires
}

// checkPreconditions evaluates request preconditions and reports whether a precondition
// resulted in sending StatusNotModified or StatusPreconditionFailed.
func checkPreconditions(w http.ResponseWriter, r *http.Request, etag string) (done bool, rangeHeader string) {
	// This function carefully follows RFC 7232 section 6.
	ch := checkIfMatch(r, etag)
	if ch == condFalse {
		w.WriteHeader(http.StatusPreconditionFailed)
		return true, ""
	}
	switch checkIfNoneMatch(r, etag) {
	case condFalse:
		if r.Method == "GET" || r.Method == "HEAD" {
			writeNotModified(w)
			return true, ""
		} else {
			w.WriteHeader(http.StatusPreconditionFailed)
			return true, ""
		}
	}

	rangeHeader = r.Header.Get("Range")
	if rangeHeader != "" && checkIfRange(r, etag) == condFalse {
		rangeHeader = ""
	}
	return false, rangeHeader
}

// condResult is the result of an HTTP request precondition check.
// See https://tools.ietf.org/html/rfc7232 section 3.
type condResult int

const (
	condNone condResult = iota
	condTrue
	condFalse
)

func checkIfMatch(r *http.Request, currentEtag string) condResult {
	im := r.Header.Get("If-Match")
	if im == "" {
		return condNone
	}
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return condTrue
		}
		etag, remain := scanETag(im)
		if etag == "" {
			break
		}
		if etagStrongMatch(etag, currentEtag) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

// scanETag determines if a syntactically valid ETag is present at s. If so,
// the ETag and remaining text after consuming ETag is returned. Otherwise,
// it returns "", "".
func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, "W/") {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	// ETag is either W/"text" or "text".
	// See RFC 7232 2.3.
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		// Character values allowed in ETags.
		case c == 0x21 || c >= 0x23 && c <= 0x7E || c >= 0x80:
		case c == '"':
			return s[:i+1], s[i+1:]
		default:
			return "", ""
		}
	}
	return "", ""
}

// etagStrongMatch reports whether a and b match using strong ETag comparison.
// Assumes a and b are valid ETags.
func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

func checkIfNoneMatch(r *http.Request, currentEtag string) condResult {
	inm := r.Header.Get("If-None-Match")
	if inm == "" {
		return condNone
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
			continue
		}
		if buf[0] == '*' {
			return condFalse
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, currentEtag) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

// etagWeakMatch reports whether a and b match using weak ETag comparison.
// Assumes a and b are valid ETags.
func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, "W/") == strings.TrimPrefix(b, "W/")
}

func writeNotModified(w http.ResponseWriter) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	h := w.Header()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	delete(h, "Content-Encoding")
	if h.Get("Etag") != "" {
		delete(h, "Last-Modified")
	}
	w.WriteHeader(http.StatusNotModified)
}

func checkIfRange(r *http.Request, currentEtag string) condResult {
	if r.Method != "GET" && r.Method != "HEAD" {
		return condNone
	}
	ir := r.Header.Get("If-Range")
	if ir == "" {
		return condNone
	}
	etag, _ := scanETag(ir)
	if etag != "" {
		if etagStrongMatch(etag, currentEtag) {
			return condTrue
		} else {
			return condFalse
		}
	}

	return condFalse
}

// httpRange specifies the byte range to be sent to the client.
type httpRange struct {
	start, length int64
}

func (r httpRange) contentRange(size int64) string {
	return fmt.Sprintf("bytes %d-%d/%d", r.start, r.start+r.length-1, size)
}

func (r httpRange) mimeHeader(contentType string, size int64) textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Range": {r.contentRange(size)},
		"Content-Type":  {contentType},
	}
}

// parseRange parses a Range header string as per RFC 7233.
// errNoOverlap is returned if none of the ranges overlap.
func parseRange(s string, size int64) ([]httpRange, error) {
	if s == "" {
		return nil, nil // header not present
	}
	const b = "bytes="
	if !strings.HasPrefix(s, b) {
		return nil, errors.New("invalid range")
	}
	var ranges []httpRange
	noOverlap := false
	for _, ra := range strings.Split(s[len(b):], ",") {
		ra = textproto.TrimString(ra)
		if ra == "" {
			continue
		}
		start, end, ok := strings.Cut(ra, "-")
		if !ok {
			return nil, errors.New("invalid range")
		}
		start, end = textproto.TrimString(start), textproto.TrimString(end)
		var r httpRange
		if start == "" {
			// If no start is specified, end specifies the
			// range start relative to the end of the file,
			// and we are dealing with <suffix-length>
			// which has to be a non-negative integer as per
			// RFC 7233 Section 2.1 "Byte-Ranges".
			if end == "" || end[0] == '-' {
				return nil, errors.New("invalid range")
			}
			i, err := strconv.ParseInt(end, 10, 64)
			if i < 0 || err != nil {
				return nil, errors.New("invalid range")
			}
			if i > size {
				i = size
			}
			r.start = size - i
			r.length = size - r.start
		} else {
			i, err := strconv.ParseInt(start, 10, 64)
			if err != nil || i < 0 {
				return nil, errors.New("invalid range")
			}
			if i >= size {
				// If the range begins after the size of the content,
				// then it does not overlap.
				noOverlap = true
				continue
			}
			r.start = i
			if end == "" {
				// If no end is specified, range extends to end of the file.
				r.length = size - r.start
			} else {
				i, err := strconv.ParseInt(end, 10, 64)
				if err != nil || r.start > i {
					return nil, errors.New("invalid range")
				}
				if i >= size {
					i = size - 1
				}
				r.length = i - r.start + 1
			}
		}
		ranges = append(ranges, r)
	}
	if noOverlap && len(ranges) == 0 {
		// The specified ranges did not overlap with the content.
		return nil, errNoOverlap
	}
	return ranges, nil
}

func sumRangesSize(ranges []httpRange) (size int64) {
	for _, ra := range ranges {
		size += ra.length
	}
	return
}

// parseContentRangeOffset parses the start offset from a Content-Range header.
// Content-Range format: "bytes start-end/total" (e.g., "bytes 100-200/1000")
// Returns 0 if the header is empty, invalid, or cannot be parsed.
func parseContentRangeOffset(contentRange string) int64 {
	if contentRange == "" {
		return 0
	}

	// Content-Range format: "bytes start-end/total"
	if !strings.HasPrefix(contentRange, "bytes ") {
		return 0
	}

	rangeSpec := strings.TrimPrefix(contentRange, "bytes ")
	dashPos := strings.Index(rangeSpec, "-")
	if dashPos <= 0 {
		return 0
	}

	start, err := strconv.ParseInt(rangeSpec[:dashPos], 10, 64)
	if err != nil {
		return 0
	}

	return start
}

// countingWriter counts how many bytes have been written to it.
type countingWriter int64

func (w *countingWriter) Write(p []byte) (n int, err error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

// rangesMIMESize returns the number of bytes it takes to encode the
// provided ranges as a multipart response.
func rangesMIMESize(ranges []httpRange, contentType string, contentSize int64) (encSize int64) {
	var w countingWriter
	mw := multipart.NewWriter(&w)
	for _, ra := range ranges {
		mw.CreatePart(ra.mimeHeader(contentType, contentSize))
		encSize += ra.length
	}
	mw.Close()
	encSize += int64(w)
	return
}

type lrs struct {
	c io.Closer
	r io.Reader
}

func (r lrs) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r lrs) Close() error {
	return r.c.Close()
}
