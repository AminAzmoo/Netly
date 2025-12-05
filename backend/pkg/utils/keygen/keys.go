package keygen

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"

	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
)

// GenerateUUID generates a random UUID v4
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateShortId generates 8 random hex characters (4 bytes)
// Used for VLESS Reality ShortId
func GenerateShortId() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "deadbeef" // Fallback (should never happen)
	}
	return fmt.Sprintf("%x", b)
}

// GenerateRandomPassword generates a secure random string of given length
func GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

// GenerateX25519Keys generates Curve25519 keypair for Xray/Sing-box Reality
// Returns Base64URL encoded strings (without padding)
func GenerateX25519Keys() (string, string, error) {
	var privateKey [32]byte
	if _, err := io.ReadFull(rand.Reader, privateKey[:]); err != nil {
		return "", "", err
	}

	// Curve25519 private key clamping (standard procedure)
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	// Xray/Sing-box uses RawURLEncoding (no padding)
	privStr := base64.RawURLEncoding.EncodeToString(privateKey[:])
	pubStr := base64.RawURLEncoding.EncodeToString(publicKey[:])

	return privStr, pubStr, nil
}

// GenerateWireGuardKeys generates standard WireGuard keys
// Returns standard Base64 encoded strings
func GenerateWireGuardKeys() (string, string, error) {
	var privateKey [32]byte
	if _, err := io.ReadFull(rand.Reader, privateKey[:]); err != nil {
		return "", "", err
	}

	// Curve25519 private key clamping
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	// WireGuard uses Standard Encoding
	privStr := base64.StdEncoding.EncodeToString(privateKey[:])
	pubStr := base64.StdEncoding.EncodeToString(publicKey[:])

	return privStr, pubStr, nil
}
