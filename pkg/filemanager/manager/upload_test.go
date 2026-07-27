package manager

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"strings"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/encrypt"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticMasterKeyVault struct {
	key []byte
}

func (v staticMasterKeyVault) GetMasterKey(context.Context) ([]byte, error) {
	return v.key, nil
}

type readSeekCloser struct {
	*bytes.Reader
}

func (readSeekCloser) Close() error {
	return nil
}

func testCryptorFactory(t *testing.T) encrypt.CryptorFactory {
	t.Helper()

	masterKey := make([]byte, 32)
	_, err := rand.Read(masterKey)
	require.NoError(t, err)

	return encrypt.NewCryptorFactory(staticMasterKeyVault{key: masterKey})
}

// persistedMetadata returns encryption metadata in the shape it is stored on an entity, that is
// with the plaintext data key dropped, so that tests exercise the unwrapping path.
func persistedMetadata(t *testing.T, factory encrypt.CryptorFactory) *types.EncryptMetadata {
	t.Helper()

	cryptor, err := factory(types.CipherAES256CTR)
	require.NoError(t, err)

	generated, err := cryptor.GenerateMetadata(context.Background())
	require.NoError(t, err)

	return &types.EncryptMetadata{
		Algorithm: generated.Algorithm,
		Key:       generated.Key,
		IV:        generated.IV,
	}
}

func decryptBlob(t *testing.T, factory encrypt.CryptorFactory, metadata *types.EncryptMetadata,
	ciphertext []byte, counterOffset int64) []byte {
	t.Helper()

	cryptor, err := factory(metadata.Algorithm)
	require.NoError(t, err)
	require.NoError(t, cryptor.LoadMetadata(context.Background(), metadata))
	require.NoError(t, cryptor.SetSource(readSeekCloser{bytes.NewReader(ciphertext)}, nil,
		int64(len(ciphertext)), counterOffset))

	plaintext, err := io.ReadAll(cryptor)
	require.NoError(t, err)

	return plaintext
}

func newTestUploadRequest(plaintext []byte) *fs.UploadRequest {
	src := readSeekCloser{bytes.NewReader(plaintext)}
	return &fs.UploadRequest{
		Props:  &fs.UploadProps{Size: int64(len(plaintext))},
		File:   src,
		Seeker: src,
	}
}

// A blob encrypted on its way into a storage policy has to decrypt back to the original bytes,
// and has to keep its length: the size recorded on the entity is the plaintext size, and storage
// drivers write exactly that many bytes.
func TestEncryptUploadRequestRoundTripPreservesLength(t *testing.T) {
	factory := testCryptorFactory(t)
	metadata := persistedMetadata(t, factory)
	plaintext := []byte(strings.Repeat("cloudreve relocation payload ", 200))

	req := newTestUploadRequest(plaintext)
	require.NoError(t, encryptUploadRequest(context.Background(), factory, req, metadata, 0))

	ciphertext, err := io.ReadAll(req)
	require.NoError(t, err)

	assert.Len(t, ciphertext, len(plaintext))
	assert.NotEqual(t, plaintext, ciphertext)
	assert.Equal(t, plaintext, decryptBlob(t, factory, metadata, ciphertext, 0))
}

// Chunked storage drivers rewind the upload request to replay a chunk on retry. Rewinding the
// plaintext without rewinding the keystream would silently produce corrupted ciphertext, and
// nothing downstream computes a checksum that would catch it.
func TestEncryptUploadRequestRewindOnRetryIsStable(t *testing.T) {
	factory := testCryptorFactory(t)
	metadata := persistedMetadata(t, factory)
	plaintext := []byte(strings.Repeat("retried chunk ", 300))

	req := newTestUploadRequest(plaintext)
	require.NoError(t, encryptUploadRequest(context.Background(), factory, req, metadata, 0))

	// Abandon a partially read chunk, the way a failed request does.
	discarded := make([]byte, 64)
	_, err := io.ReadFull(req, discarded)
	require.NoError(t, err)

	_, err = req.Seek(0, io.SeekStart)
	require.NoError(t, err)

	replayed, err := io.ReadAll(req)
	require.NoError(t, err)

	assert.Len(t, replayed, len(plaintext))
	assert.Equal(t, plaintext, decryptBlob(t, factory, metadata, replayed, 0))
}

// A non-zero counter offset keeps the keystream aligned when only a part of a blob is written, so
// that a chunk written at an offset stays decryptable as part of the whole blob.
func TestEncryptUploadRequestCounterOffsetAlignsKeystream(t *testing.T) {
	const offset = 512

	factory := testCryptorFactory(t)
	metadata := persistedMetadata(t, factory)
	plaintext := []byte(strings.Repeat("offset aligned payload ", 100))
	require.Greater(t, len(plaintext), offset)

	whole := newTestUploadRequest(plaintext)
	require.NoError(t, encryptUploadRequest(context.Background(), factory, whole, metadata, 0))
	full, err := io.ReadAll(whole)
	require.NoError(t, err)

	tail := newTestUploadRequest(plaintext[offset:])
	require.NoError(t, encryptUploadRequest(context.Background(), factory, tail, metadata, offset))
	partial, err := io.ReadAll(tail)
	require.NoError(t, err)

	assert.Equal(t, full[offset:], partial)
}
