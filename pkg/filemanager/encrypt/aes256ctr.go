// Package encrypt provides AES-256-CTR encryption and decryption functionality
// compatible with the JavaScript EncryptedBlob implementation.
//
// # Usage Example
//
// Basic usage with encrypted metadata:
//
//	// Create AES256CTR instance
//	aes := NewAES256CTR(masterKeyVault)
//
//	// Load encrypted metadata (key is encrypted with master key)
//	err := aes.LoadMetadata(ctx, encryptedMetadata, masterKeyVault)
//	if err != nil {
//	    return err
//	}
//
//	// Set encrypted source stream
//	err = aes.SetSource(encryptedStream, 0)
//	if err != nil {
//	    return err
//	}
//
//	// Read decrypted data
//	decryptedData, err := io.ReadAll(aes)
//	if err != nil {
//	    return err
//	}
//	aes.Close()
//
// Usage with plain metadata (already decrypted):
//
//	aes := NewAES256CTR(masterKeyVault)
//	err := aes.LoadPlainMetadata(plainMetadata)
//	err = aes.SetSource(encryptedStream, 0)
//	// Read decrypted data...
//
// Usage with counter offset (for chunked/sliced streams):
//
//	// If reading from byte offset 1048576 (1MB) of the encrypted file
//	aes := NewAES256CTR(masterKeyVault)
//	err := aes.LoadPlainMetadata(metadata)
//	err = aes.SetSource(encryptedStreamStartingAt1MB, 1048576)
//	// This ensures proper counter alignment for correct decryption
//
// Using the Seeker interface (requires seekable source):
//
//	aes := NewAES256CTR(masterKeyVault)
//	err := aes.LoadPlainMetadata(metadata)
//	err = aes.SetSource(seekableEncryptedStream, 0)
//	aes.SetSize(totalFileSize) // Required for io.SeekEnd
//
//	// Seek to position 1048576
//	newPos, err := aes.Seek(1048576, io.SeekStart)
//	// Read from that position...
//
//	// Seek relative to current position
//	newPos, err = aes.Seek(100, io.SeekCurrent)
//
//	// Seek from end (requires SetSize to be called first)
//	newPos, err = aes.Seek(-1024, io.SeekEnd)
//
// Using the factory pattern:
//
//	factory := NewDecrypterFactory(masterKeyVault)
//	decrypter, err := factory(types.CipherAES256CTR)
//	if err != nil {
//	    return err
//	}
//	err = decrypter.LoadMetadata(ctx, encryptedMetadata, masterKeyVault)
//	err = decrypter.SetSource(encryptedStream, 0)
//	defer decrypter.Close()
package encrypt

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// AES256CTR provides both encryption and decryption for AES-256-CTR.
// It implements both Cryptor and Decrypter interfaces.
type AES256CTR struct {
	masterKeyVault MasterEncryptKeyVault

	// Decryption fields
	src           io.ReadCloser // Source encrypted stream
	seeker        io.Seeker     // Seeker for the source stream
	stream        cipher.Stream // AES-CTR cipher stream
	metadata      *types.EncryptMetadata
	counterOffset int64 // Byte offset for sliced streams
	pos           int64 // Current read position relative to counterOffset
	size          int64 // Total size of encrypted data (for SeekEnd support, -1 if unknown)
	eof           bool  // EOF flag
}

func NewAES256CTR(masterKeyVault MasterEncryptKeyVault) *AES256CTR {
	return &AES256CTR{
		masterKeyVault: masterKeyVault,
		size:           -1, // Unknown by default
	}
}

func (e *AES256CTR) GenerateMetadata(ctx context.Context) (*types.EncryptMetadata, error) {
	// Generate random 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	// Generate random 16-byte IV for CTR mode
	iv := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Get master key from vault
	masterKey, err := e.masterKeyVault.GetMasterKey(ctx)
	if err != nil {
		return nil, err
	}

	// Encrypt the key with master key
	encryptedKey, err := EncryptWithMasterKey(masterKey, key)
	if err != nil {
		return nil, err
	}

	return &types.EncryptMetadata{
		Algorithm:    types.CipherAES256CTR,
		Key:          encryptedKey,
		KeyPlainText: key,
		IV:           iv,
	}, nil
}

// LoadMetadata loads and decrypts the encryption metadata using the master key.
func (e *AES256CTR) LoadMetadata(ctx context.Context, encryptedMetadata *types.EncryptMetadata) error {
	if encryptedMetadata == nil {
		return fmt.Errorf("encryption metadata is nil")
	}

	if encryptedMetadata.Algorithm != types.CipherAES256CTR {
		return fmt.Errorf("unsupported algorithm: %s", encryptedMetadata.Algorithm)
	}

	if len(encryptedMetadata.KeyPlainText) > 0 {
		e.metadata = encryptedMetadata
		return nil
	}

	// Decrypt the encryption key
	decryptedKey, err := DecriptKey(ctx, e.masterKeyVault, encryptedMetadata.Key)
	if err != nil {
		return fmt.Errorf("failed to decrypt encryption key: %w", err)
	}

	// Store decrypted metadata
	e.metadata = &types.EncryptMetadata{
		Algorithm:    encryptedMetadata.Algorithm,
		KeyPlainText: decryptedKey,
		IV:           encryptedMetadata.IV,
	}

	return nil
}

// SetSource sets the encrypted data source and initializes the cipher stream.
// The counterOffset parameter allows for proper decryption of sliced streams,
// where the stream doesn't start at byte 0 of the original file.
//
// For non-block-aligned offsets (offset % 16 != 0), this method advances the
// cipher stream to the correct position within the block to ensure proper decryption.
func (e *AES256CTR) SetSource(src io.ReadCloser, seeker io.Seeker, size, counterOffset int64) error {
	if e.metadata == nil {
		return fmt.Errorf("metadata not loaded, call LoadMetadata first")
	}

	e.src = src
	e.seeker = seeker
	e.counterOffset = counterOffset
	e.pos = 0     // Reset position to start
	e.eof = false // Reset EOF flag
	e.size = size

	// Initialize cipher stream at counterOffset position
	return e.initCipherStream(counterOffset)
}

// Read implements io.Reader interface to read decrypted data.
// It reads encrypted data from the source and decrypts it on-the-fly.
func (e *AES256CTR) Read(p []byte) (int, error) {
	if e.src == nil {
		return 0, fmt.Errorf("source not set, call SetSource first")
	}

	if e.eof {
		return 0, io.EOF
	}

	// Read encrypted data from source
	n, err := e.src.Read(p)
	if err != nil {
		if err == io.EOF {
			e.eof = true
			if n == 0 {
				return 0, io.EOF
			}
		} else {
			return n, err
		}
	}

	// Decrypt data in place
	if n > 0 {
		e.stream.XORKeyStream(p[:n], p[:n])
		e.pos += int64(n) // Update current position
	}

	return n, err
}

// Close implements io.Closer interface.
func (e *AES256CTR) Close() error {
	if e.src != nil {
		return e.src.Close()
	}
	return nil
}

// Seek implements io.Seeker interface for seeking within the encrypted stream.
// It properly adjusts the AES-CTR counter based on the seek position.
//
// Parameters:
//   - offset: byte offset relative to whence
//   - whence: io.SeekStart, io.SeekCurrent, or io.SeekEnd
//
// Returns the new absolute position (relative to counterOffset start).
//
// Note: For io.SeekEnd to work, you must call SetSize() first, otherwise it returns an error.
// Also note that seeking requires the underlying source to support seeking (io.Seeker).
func (e *AES256CTR) Seek(offset int64, whence int) (int64, error) {
	if e.metadata == nil {
		return 0, fmt.Errorf("metadata not loaded, call LoadMetadata first")
	}

	if e.src == nil {
		return 0, fmt.Errorf("source not set, call SetSource first")
	}

	// Check if source supports seeking
	if e.seeker == nil {
		return 0, fmt.Errorf("source does not support seeking")
	}

	// Calculate new absolute position
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = e.pos + offset
	case io.SeekEnd:
		if e.size < 0 {
			return 0, fmt.Errorf("size unknown, call SetSize before using SeekEnd")
		}
		newPos = e.size + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	// Validate new position
	if newPos < 0 {
		return 0, fmt.Errorf("negative position: %d", newPos)
	}

	// Seek in the underlying source stream
	// The absolute position in the source is counterOffset + newPos
	absPos := e.counterOffset + newPos
	_, err := e.seeker.Seek(absPos, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("failed to seek source: %w", err)
	}

	// Reinitialize cipher stream with new counter position
	if err := e.initCipherStream(absPos); err != nil {
		return 0, fmt.Errorf("failed to reinitialize cipher stream: %w", err)
	}

	// Update position and reset EOF flag
	e.pos = newPos
	e.eof = false

	return newPos, nil
}

// initCipherStream initializes the cipher stream with proper counter alignment
// for the given absolute byte position.
func (e *AES256CTR) initCipherStream(absolutePosition int64) error {
	// Create AES cipher block
	block, err := aes.NewCipher(e.metadata.KeyPlainText)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create counter value (16 bytes IV) and apply offset for position
	counter := make([]byte, 16)
	copy(counter, e.metadata.IV)

	// Apply counter offset based on byte position (each block is 16 bytes)
	if absolutePosition > 0 {
		blockOffset := absolutePosition / 16
		incrementCounter(counter, blockOffset)
	}

	// Create CTR cipher stream
	e.stream = cipher.NewCTR(block, counter)

	// For non-block-aligned offsets, we need to advance the stream position
	// within the current block to match the offset
	offsetInBlock := absolutePosition % 16
	if offsetInBlock > 0 {
		// Create a dummy buffer to advance the stream
		dummy := make([]byte, offsetInBlock)
		e.stream.XORKeyStream(dummy, dummy)
	}

	return nil
}

// incrementCounter increments a counter ([]byte) by a given number of blocks.
// This matches the JavaScript implementation's incrementCounter function.
// The counter is treated as a big-endian 128-bit integer.
func incrementCounter(counter []byte, blocks int64) {
	// Convert blocks to add into bytes (big-endian)
	// We only need to handle the lower 64 bits since blocks is int64
	for i := 15; i >= 0 && blocks > 0; i-- {
		// Add the lowest byte of blocks to current counter byte
		sum := uint64(counter[i]) + uint64(blocks&0xff)
		counter[i] = byte(sum & 0xff)

		// Shift blocks right by 8 bits for next iteration
		blocks = blocks >> 8

		// Add carry from this position to the next
		if sum > 0xff {
			carry := sum >> 8
			// Propagate carry to higher bytes
			for j := i - 1; j >= 0 && carry > 0; j-- {
				sum = uint64(counter[j]) + carry
				counter[j] = byte(sum & 0xff)
				carry = sum >> 8
			}
		}
	}
}
