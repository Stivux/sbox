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

	saltLen        = 16
	nonceLen       = 12
	gcmTagLen      = 16
	encryptedMagic = "SBOXE1"
	headerLen      = len(encryptedMagic) + saltLen + nonceLen
)

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
	salt, err := randN(saltLen)
	if err != nil {
		return nil, fmt.Errorf("%w: salt generation: %w", ErrEncrypt, err)
	}
	gcm, cleanup, err := gcmFromPassword(password, salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncrypt, err)
	}
	defer cleanup()

	nonce, err := randN(nonceLen)
	if err != nil {
		return nil, fmt.Errorf("%w: nonce generation: %w", ErrEncrypt, err)
	}
	out := make([]byte, 0, headerLen+len(plaintext)+gcm.Overhead())
	out = append(out, encryptedMagic...)
	out = append(out, salt...)
	out = append(out, nonce...)
	// Header is AAD: a swapped salt or nonce fails authentication.
	return gcm.Seal(out, nonce, plaintext, out[:headerLen]), nil
}

// Decrypt opens an Encrypt payload with the given password.
func Decrypt(data []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}
	if !IsEncrypted(data) {
		return nil, ErrNotEncrypted
	}
	if len(data) < headerLen+gcmTagLen {
		return nil, ErrCorruptPayload
	}
	salt := data[len(encryptedMagic) : len(encryptedMagic)+saltLen]
	nonce := data[len(encryptedMagic)+saltLen : headerLen]
	gcm, cleanup, err := gcmFromPassword(password, salt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecrypt, err)
	}
	defer cleanup()

	plaintext, err := gcm.Open(nil, nonce, data[headerLen:], data[:headerLen])
	if err != nil {
		return nil, ErrWrongPassword
	}
	return plaintext, nil
}

func randN(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, b)
	return b, err
}

func gcmFromPassword(password string, salt []byte) (cipher.AEAD, func(), error) {
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	cleanup := func() { clear(key) }
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
