package core

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32

	saltLen   = 16
	nonceLen  = 12
	gcmTagLen = 16
)

const encryptedMagic string = "SBOXE1"

var (
	ErrEmptyPassword  = errors.New("password required")
	ErrEncrypt        = errors.New("encrypt failed")
	ErrDecrypt        = errors.New("decrypt failed")
	ErrWrongPassword  = errors.New("wrong password or corrupted data")
	ErrNotEncrypted   = errors.New("data is not encrypted")
	ErrCorruptPayload = errors.New("corrupt encrypted payload")
)

// IsEncrypted reports whether data starts with the encrypted vault magic header.
func IsEncrypted(data []byte) bool {
	return bytes.HasPrefix(data, []byte(encryptedMagic))
}

// Encrypt seals plaintext with AES-256-GCM keyed by Argon2id(password).
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("%w: salt generation: %w", ErrEncrypt, err)
	}

	gcm, cleanupKey, err := gcmFromPassword(password, salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncrypt, err)
	}
	defer cleanupKey()

	nonce := make([]byte, nonceLen)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: nonce generation: %w", ErrEncrypt, err)
	}

	headerLen := len(encryptedMagic) + saltLen + nonceLen
	out := make([]byte, 0, headerLen+len(plaintext)+gcm.Overhead())
	out = append(out, encryptedMagic...)
	out = append(out, salt...)
	out = append(out, nonce...)

	// Pass the header slice as Associated Authenticated Data (AAD)
	aad := out[:headerLen]
	return gcm.Seal(out, nonce, plaintext, aad), nil
}

// Decrypt opens an Encrypt payload with the given password.
func Decrypt(data []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}
	if !IsEncrypted(data) {
		return nil, ErrNotEncrypted
	}

	headerLen := len(encryptedMagic) + saltLen + nonceLen
	if len(data) < headerLen+gcmTagLen {
		return nil, ErrCorruptPayload
	}

	aad := data[:headerLen]
	offset := len(encryptedMagic)
	salt := data[offset : offset+saltLen]
	offset += saltLen
	nonce := data[offset : offset+nonceLen]
	ciphertext := data[headerLen:]

	gcm, cleanupKey, err := gcmFromPassword(password, salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecrypt, err)
	}
	defer cleanupKey()

	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrWrongPassword
	}
	return plaintext, nil
}

func gcmFromPassword(password string, salt []byte) (cipher.AEAD, func(), error) {
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	cleanup := func() {
		clear(key)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	return gcm, cleanup, nil
}
