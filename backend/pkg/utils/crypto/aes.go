package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidKey        = errors.New("crypto: invalid encryption key")
	ErrEncryptionFailed  = errors.New("crypto: encryption failed")
	ErrDecryptionFailed  = errors.New("crypto: decryption failed")
	ErrInvalidCipherText = errors.New("crypto: invalid cipher text")
)

// deriveKey creates a 32-byte key from any string using SHA-256
func deriveKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

// Encrypt encrypts plaintext using AES-256-GCM
func Encrypt(plainText string, key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	derivedKey := deriveKey(key)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", ErrEncryptionFailed
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrEncryptionFailed
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrEncryptionFailed
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(cipherText string, key string) (string, error) {
	if key == "" {
		return "", ErrInvalidKey
	}

	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", ErrInvalidCipherText
	}

	derivedKey := deriveKey(key)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCipherText
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plainText), nil
}
