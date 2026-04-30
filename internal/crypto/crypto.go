package crypto

import (
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/box"
)

// SealedBox provides encryption/decryption using NaCl box (Curve25519 + XSalsa20-Poly1305).
type SealedBox struct {
	secretKey *[32]byte
	publicKey *[32]byte
}

// GenerateKey generates a new key pair for sealed box encryption.
func GenerateKey() (publicKey, secretKey *[32]byte, err error) {
	return box.GenerateKey(rand.Reader)
}

// NewSealedBox creates a SealedBox with the given key pair.
func NewSealedBox(publicKey, secretKey *[32]byte) *SealedBox {
	return &SealedBox{
		secretKey: secretKey,
		publicKey: publicKey,
	}
}

// NewSealedBoxFromSecret creates a SealedBox from a secret key seed.
// The secret key is used to derive both keys (for symmetric usage).
func NewSealedBoxFromSecret(secret *[32]byte) *SealedBox {
	// For symmetric encryption, we derive the public key from the secret key.
	// In a sealed box setup, the recipient's key pair is used.
	// For our use case (self-encryption), we use the same key pair.
	publicKey := &[32]byte{}
	secretKey := &[32]byte{}
	copy(publicKey[:], secret[:])
	copy(secretKey[:], secret[:])
	return &SealedBox{
		secretKey: secretKey,
		publicKey: publicKey,
	}
}

// Encrypt seals a message using the public key (anonymous encryption).
// Only the holder of the corresponding secret key can decrypt it.
func (s *SealedBox) Encrypt(plaintext []byte) ([]byte, error) {
	// Sealed box: anonymous sender, recipient with secret key can open
	encrypted, err := box.SealAnonymous(nil, plaintext, s.publicKey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("seal anonymous: %w", err)
	}
	return encrypted, nil
}

// Decrypt opens a sealed box using the secret key.
func (s *SealedBox) Decrypt(ciphertext []byte) ([]byte, error) {
	decrypted, ok := box.OpenAnonymous(nil, ciphertext, s.publicKey, s.secretKey)
	if !ok {
		return nil, fmt.Errorf("open anonymous: decryption failed")
	}
	return decrypted, nil
}

// GenerateRandomKey creates a cryptographically random 32-byte key.
func GenerateRandomKey() (*[32]byte, error) {
	key := &[32]byte{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		return nil, fmt.Errorf("generate random key: %w", err)
	}
	return key, nil
}

// KeyToBytes converts a 32-byte key to a byte slice.
func KeyToBytes(key *[32]byte) []byte {
	return key[:]
}

// BytesToKey converts a 32-byte slice to a 32-byte array.
func BytesToKey(b []byte) (*[32]byte, error) {
	if len(b) != 32 {
		return nil, fmt.Errorf("invalid key length: %d (expected 32)", len(b))
	}
	key := &[32]byte{}
	copy(key[:], b)
	return key, nil
}
