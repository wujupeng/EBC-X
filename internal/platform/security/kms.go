package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

var (
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrKeyNotFound       = errors.New("encryption key not found")
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
)

type EncryptionKey struct {
	KeyID      string
	Key        []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	IsActive   bool
}

type KMS struct {
	mu      sync.RWMutex
	keys    map[string]*EncryptionKey
	active  string
}

func NewKMS() *KMS {
	return &KMS{
		keys: make(map[string]*EncryptionKey),
	}
}

func (k *KMS) GenerateKey(keyID string) error {
	keyBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	now := time.Now()
	key := &EncryptionKey{
		KeyID:     keyID,
		Key:       keyBytes,
		CreatedAt: now,
		ExpiresAt: now.Add(365 * 24 * time.Hour),
		IsActive:  true,
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	for _, existing := range k.keys {
		if existing.IsActive {
			existing.IsActive = false
		}
	}
	k.keys[keyID] = key
	k.active = keyID
	return nil
}

func (k *KMS) GetActiveKey() (*EncryptionKey, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.active == "" {
		return nil, ErrKeyNotFound
	}
	key, exists := k.keys[k.active]
	if !exists || !key.IsActive {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

func (k *KMS) GetKey(keyID string) (*EncryptionKey, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	key, exists := k.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

func (k *KMS) RotateKey(newKeyID string) error {
	return k.GenerateKey(newKeyID)
}

func (k *KMS) Encrypt(plaintext []byte) (string, error) {
	key, err := k.GetActiveKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%s:%s", key.KeyID, hex.EncodeToString(ciphertext)), nil
}

func (k *KMS) Decrypt(encrypted string) ([]byte, error) {
	idx := strings.Index(encrypted, ":")
	if idx < 0 {
		return nil, ErrInvalidCiphertext
	}
	keyID := encrypted[:idx]
	hexCiphertext := encrypted[idx+1:]

	key, err := k.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	ciphertext, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertextBody := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func (k *KMS) EncryptField(plaintext string) (string, error) {
	return k.Encrypt([]byte(plaintext))
}

func (k *KMS) DecryptField(encrypted string) (string, error) {
	plaintext, err := k.Decrypt(encrypted)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (k *KMS) HashPII(plaintext string) string {
	h := sha256.New()
	h.Write([]byte(plaintext))
	return hex.EncodeToString(h.Sum(nil))
}

func (k *KMS) ActiveKeyID() string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.active
}