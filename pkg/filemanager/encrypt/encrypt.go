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

type (
	Cryptor interface {
		io.ReadCloser
		io.Seeker
		// LoadMetadata loads and decrypts the encryption metadata using the master key
		LoadMetadata(ctx context.Context, encryptedMetadata *types.EncryptMetadata) error
		// SetSource sets the encrypted data source and initializes the cipher stream
		SetSource(src io.ReadCloser, seeker io.Seeker, size, counterOffset int64) error
		// GenerateMetadata generates a new encryption metadata
		GenerateMetadata(ctx context.Context) (*types.EncryptMetadata, error)
	}

	CryptorFactory func(algorithm types.Cipher) (Cryptor, error)
)

func NewCryptorFactory(masterKeyVault MasterEncryptKeyVault) CryptorFactory {
	return func(algorithm types.Cipher) (Cryptor, error) {
		switch algorithm {
		case types.CipherAES256CTR:
			return NewAES256CTR(masterKeyVault), nil
		default:
			return nil, fmt.Errorf("unknown algorithm: %s", algorithm)
		}
	}
}

// EncryptWithMasterKey encrypts data using the master key with AES-256-CTR
// Returns: [16-byte IV] + [encrypted data]
func EncryptWithMasterKey(masterKey, data []byte) ([]byte, error) {
	// Create AES cipher with master key
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}

	// Generate random IV for encryption
	iv := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Encrypt data
	stream := cipher.NewCTR(block, iv)
	encrypted := make([]byte, len(data))
	stream.XORKeyStream(encrypted, data)

	// Return IV + encrypted data
	result := append(iv, encrypted...)
	return result, nil
}

func DecriptKey(ctx context.Context, keyVault MasterEncryptKeyVault, encryptedKey []byte) ([]byte, error) {
	masterKey, err := keyVault.GetMasterKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}
	return DecryptWithMasterKey(masterKey, encryptedKey)
}

// DecryptWithMasterKey decrypts data using the master key with AES-256-CTR
// Input format: [16-byte IV] + [encrypted data]
func DecryptWithMasterKey(masterKey, encryptedData []byte) ([]byte, error) {
	// Validate input length
	if len(encryptedData) < 16 {
		return nil, aes.KeySizeError(len(encryptedData))
	}

	// Extract IV and encrypted data
	iv := encryptedData[:16]
	encrypted := encryptedData[16:]

	// Create AES cipher with master key
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}

	// Decrypt data
	stream := cipher.NewCTR(block, iv)
	decrypted := make([]byte, len(encrypted))
	stream.XORKeyStream(decrypted, encrypted)

	return decrypted, nil
}
