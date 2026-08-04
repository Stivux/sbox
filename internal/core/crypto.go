package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// Encrypted subscription file layout:
//
//	magic (6) | salt (16) | nonce (12) | ciphertext+tag
const (
	encryptedMagic = "SBOXE1"

	argonTime    = 3
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 4
	argonKeyLen  = 32

	saltLen  = 16
	nonceLen = 12 // AES-GCM standard nonce size
)

var (
	ErrEmptyPassword  = errors.New("password required")
	ErrEncrypt        = errors.New("encrypt failed")
	ErrDecrypt        = errors.New("decrypt failed")
	ErrWrongPassword  = errors.New("wrong password or corrupted data")
	ErrNotEncrypted   = errors.New("data is not encrypted")
	ErrCorruptPayload = errors.New("corrupt encrypted payload")
)

// IsEncrypted reports whether data starts with the encrypted subscription magic.
func IsEncrypted(data []byte) bool {
	return len(data) >= len(encryptedMagic) && string(data[:len(encryptedMagic)]) == encryptedMagic
}

// Encrypt derives a key from password via Argon2id and seals plaintext with AES-256-GCM.
// Returns magic|salt|nonce|ciphertext.
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("%w: salt: %w", ErrEncrypt, err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncrypt, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncrypt, err)
	}

	nonce := make([]byte, nonceLen)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: nonce: %w", ErrEncrypt, err)
	}

	// Preallocate exact total length (Header + Plaintext + Tag)
	out := make([]byte, 0, len(encryptedMagic)+saltLen+nonceLen+len(plaintext)+gcm.Overhead())
	out = append(out, encryptedMagic...)
	out = append(out, salt...)
	out = append(out, nonce...)

	// gcm.Seal appends the ciphertext directly into 'out'
	return gcm.Seal(out, nonce, plaintext, nil), nil
}

// Decrypt opens an Encrypt payload with the given password.
func Decrypt(data []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	if !IsEncrypted(data) {
		return nil, ErrNotEncrypted
	}

	const (
		gcmTagLen = 16
		headerLen = len(encryptedMagic) + saltLen + nonceLen
	)

	if len(data) < headerLen+gcmTagLen {
		return nil, ErrCorruptPayload
	}

	offset := len(encryptedMagic)
	salt := data[offset : offset+saltLen]
	offset += saltLen
	nonce := data[offset : offset+nonceLen]
	offset += nonceLen
	ciphertext := data[offset:]

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecrypt, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecrypt, err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrWrongPassword
	}

	return plaintext, nil
}
