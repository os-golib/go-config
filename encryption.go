package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// Encryptor defines an interface for encrypting and decrypting strings.
type Encryptor interface {
	Encrypt(value string) (string, error)
	Decrypt(encryptedValue string) (string, error)
}

// AESEncryptor implements AES-GCM encryption.
type AESEncryptor struct {
	gcm cipher.AEAD
}

// NewAESEncryptor creates a new AESEncryptor using a key string.
// The key is hashed using SHA256 to ensure it's 32 bytes for AES-256.
func NewAESEncryptor(key string) (*AESEncryptor, error) {
	// Hash the key to get a 32-byte key for AES-256
	hasher := sha256.New()
	hasher.Write([]byte(key))
	keyBytes := hasher.Sum(nil)

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AESEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts a value and returns a base64-encoded string.
func (e *AESEncryptor) Encrypt(value string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := e.gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded string.
func (e *AESEncryptor) Decrypt(encryptedValue string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("decoding base64: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting ciphertext: %w", err)
	}

	return string(plaintext), nil
}

// EncryptionProcessor processes configuration maps, decrypting values with a specific prefix.
type EncryptionProcessor struct {
	encryptor Encryptor
	prefix    string
}

// NewEncryptionProcessor creates a new processor.
// The prefix identifies which string values should be decrypted.
func NewEncryptionProcessor(encryptor Encryptor, prefix string) *EncryptionProcessor {
	return &EncryptionProcessor{
		encryptor: encryptor,
		prefix:    prefix,
	}
}

// Process recursively processes a map, decrypting any string values with the configured prefix.
func (ep *EncryptionProcessor) Process(data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for key, value := range data {
		processed, err := ep.processValue(value)
		if err != nil {
			return nil, fmt.Errorf("processing key %q: %w", key, err)
		}
		result[key] = processed
	}
	return result, nil
}

// processValue recursively processes a value.
func (ep *EncryptionProcessor) processValue(value any) (any, error) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, ep.prefix) {
			encryptedValue := strings.TrimPrefix(v, ep.prefix)
			return ep.encryptor.Decrypt(encryptedValue)
		}
		return v, nil

	case map[string]any:
		result := make(map[string]any)
		for k, val := range v {
			processed, err := ep.processValue(val)
			if err != nil {
				return nil, err
			}
			result[k] = processed
		}
		return result, nil

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			processed, err := ep.processValue(val)
			if err != nil {
				return nil, err
			}
			result[i] = processed
		}
		return result, nil

	default:
		return v, nil
	}
}

// EncryptionSource is a wrapper that applies decryption to another source.
type EncryptionSource struct {
	BaseSource
	source    Source
	processor *EncryptionProcessor
}

// NewEncryptionSource creates a new EncryptionSource.
func NewEncryptionSource(source Source, processor *EncryptionProcessor) *EncryptionSource {
	return &EncryptionSource{
		BaseSource: NewBaseSource("encryption:"+source.Name(), source.Priority()),
		source:     source,
		processor:  processor,
	}
}

// Load loads data from the underlying source and decrypts it.
func (s *EncryptionSource) Load() (map[string]any, error) {
	data, err := s.source.Load()
	if err != nil {
		return nil, err
	}
	return s.processor.Process(data)
}

// WatchPaths returns the watch paths from the underlying source.
func (s *EncryptionSource) WatchPaths() []string {
	return s.source.WatchPaths()
}
