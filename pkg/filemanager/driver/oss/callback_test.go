package oss

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/stretchr/testify/require"
)

type callbackTestBody struct {
	io.Reader
	read   int
	closed bool
}

func (b *callbackTestBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (b *callbackTestBody) Close() error {
	b.closed = true
	return nil
}

func TestVerifyCallbackSignatureBodyLimit(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	kv := cache.NewMemoStore("", logging.NewConsoleLogger(logging.LevelError))
	require.NoError(t, kv.Set(pubKeyCacheKey, pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: publicKey,
	}), 0))

	const path = "/api/v4/callback/oss/session/key"
	const payload = `{"name":"file.txt","source_name":"uploads/file.txt","size":1,"pic_info":""}`
	atLimit := payload + strings.Repeat(" ", MaxCallbackBodySize-len(payload))

	for _, test := range []struct {
		name          string
		body          string
		signedBody    string
		unknownLength bool
		wantErr       error
	}{
		{name: "valid payload", body: payload, signedBody: payload},
		{name: "at limit", body: atLimit, signedBody: atLimit},
		{name: "at limit without content length", body: atLimit, signedBody: atLimit, unknownLength: true},
		{
			name: "over limit with valid prefix signature", body: atLimit + " ", signedBody: atLimit,
			wantErr: ErrCallbackBodyTooLarge,
		},
		{
			name: "over limit without authorization or content length",
			body: atLimit + strings.Repeat(" ", 4096), unknownLength: true, wantErr: ErrCallbackBodyTooLarge,
		},
		{name: "truncated payload", body: payload[:len(payload)-1], signedBody: payload, wantErr: rsa.ErrVerification},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := &callbackTestBody{Reader: strings.NewReader(test.body)}
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.Body = body
			req.ContentLength = int64(len(test.body))
			if test.unknownLength {
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}
			if test.signedBody != "" {
				digest := md5.Sum([]byte(path + "\n" + test.signedBody))
				signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.MD5, digest[:])
				require.NoError(t, err)
				req.Header.Set("Authorization", base64.StdEncoding.EncodeToString(signature))
			}

			err := VerifyCallbackSignature(req, kv, nil)
			require.True(t, body.closed)
			require.Equal(t, min(len(test.body), MaxCallbackBodySize+1), body.read)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			restored, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Equal(t, test.body, string(restored))
		})
	}
}
