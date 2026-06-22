package ksx

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/dipockdas/keysync/internal/store"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	Format        = "keysync-share"
	Version       = 1
	FileTTL       = 10 * time.Minute
	MaxBundleSize = 1 << 20

	kdfName    = "argon2id"
	cipherName = "xchacha20-poly1305"
	kdfTime    = uint32(1)
	kdfMemory  = uint32(64 * 1024)
	kdfThreads = uint8(4)
	saltSize   = 16
)

var (
	ErrInvalidBundle      = errors.New("invalid keysync share bundle")
	ErrUnsupportedVersion = errors.New("unsupported keysync share bundle version")
	ErrExpired            = errors.New("keysync share bundle has expired")
	ErrAuthentication     = errors.New("keysync share bundle authentication failed")
)

type Secret struct {
	Name        string      `json:"name"`
	Value       string      `json:"value"`
	Scope       store.Scope `json:"scope"`
	Project     string      `json:"project,omitempty"`
	Environment string      `json:"environment,omitempty"`
}

type Payload struct {
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Secrets   []Secret  `json:"keys"`
}

type CryptoParameters struct {
	KDF       string `json:"kdf"`
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memory_kib"`
	Threads   uint8  `json:"threads"`
	Cipher    string `json:"cipher"`
}

type Envelope struct {
	Format           string           `json:"format"`
	Version          int              `json:"version"`
	Crypto           CryptoParameters `json:"crypto"`
	Salt             string           `json:"salt"`
	Nonce            string           `json:"nonce"`
	CreatedAt        time.Time        `json:"created_at"`
	ExpiresAt        time.Time        `json:"expires_at"`
	EncryptedPayload string           `json:"encrypted_payload"`
}

type Metadata struct {
	Format    string
	Version   int
	CreatedAt time.Time
	ExpiresAt time.Time
}

func Encrypt(payload Payload, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, fmt.Errorf("encrypt keysync share bundle: passphrase must not be empty")
	}
	if err := validatePayload(payload); err != nil {
		return nil, err
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encrypt keysync share bundle: encode payload: %w", err)
	}
	defer clear(plaintext)
	if len(plaintext) > MaxBundleSize {
		return nil, fmt.Errorf("%w: payload exceeds maximum size", ErrInvalidBundle)
	}

	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("encrypt keysync share bundle: generate salt: %w", err)
	}

	params := defaultCryptoParameters()
	key := deriveKey(passphrase, salt, params)
	defer clear(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("encrypt keysync share bundle: create cipher: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("encrypt keysync share bundle: generate nonce: %w", err)
	}

	envelope := Envelope{
		Format:    Format,
		Version:   Version,
		Crypto:    params,
		Salt:      base64.RawStdEncoding.EncodeToString(salt),
		Nonce:     base64.RawStdEncoding.EncodeToString(nonce),
		CreatedAt: payload.CreatedAt.UTC(),
		ExpiresAt: payload.ExpiresAt.UTC(),
	}
	aad, err := envelope.additionalData()
	if err != nil {
		return nil, fmt.Errorf("encrypt keysync share bundle: authenticate metadata: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, aad)
	envelope.EncryptedPayload = base64.RawStdEncoding.EncodeToString(ciphertext)

	bundle, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encrypt keysync share bundle: encode envelope: %w", err)
	}
	if len(bundle) > MaxBundleSize {
		return nil, fmt.Errorf("%w: bundle exceeds maximum size", ErrInvalidBundle)
	}
	return bundle, nil
}

func Inspect(bundle []byte, now time.Time) (Metadata, error) {
	envelope, err := parseEnvelope(bundle)
	if err != nil {
		return Metadata{}, err
	}
	metadata := Metadata{
		Format:    envelope.Format,
		Version:   envelope.Version,
		CreatedAt: envelope.CreatedAt,
		ExpiresAt: envelope.ExpiresAt,
	}
	if !now.UTC().Before(envelope.ExpiresAt) {
		return metadata, ErrExpired
	}
	return metadata, nil
}

func Decrypt(bundle, passphrase []byte, now time.Time) (Payload, error) {
	if len(passphrase) == 0 {
		return Payload{}, ErrAuthentication
	}
	envelope, err := parseEnvelope(bundle)
	if err != nil {
		return Payload{}, err
	}
	if !now.UTC().Before(envelope.ExpiresAt) {
		return Payload{}, ErrExpired
	}

	salt, err := decodeExact(envelope.Salt, saltSize)
	if err != nil {
		return Payload{}, ErrInvalidBundle
	}
	nonce, err := decodeExact(envelope.Nonce, chacha20poly1305.NonceSizeX)
	if err != nil {
		return Payload{}, ErrInvalidBundle
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.EncryptedPayload)
	if err != nil || len(ciphertext) < chacha20poly1305.Overhead {
		return Payload{}, ErrInvalidBundle
	}

	key := deriveKey(passphrase, salt, envelope.Crypto)
	defer clear(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return Payload{}, ErrInvalidBundle
	}
	aad, err := envelope.additionalData()
	if err != nil {
		return Payload{}, ErrInvalidBundle
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return Payload{}, ErrAuthentication
	}
	defer clear(plaintext)

	var payload Payload
	if err := decodeStrict(plaintext, &payload); err != nil {
		return Payload{}, ErrInvalidBundle
	}
	if err := validatePayload(payload); err != nil {
		return Payload{}, ErrInvalidBundle
	}
	if !payload.CreatedAt.Equal(envelope.CreatedAt) || !payload.ExpiresAt.Equal(envelope.ExpiresAt) {
		return Payload{}, ErrInvalidBundle
	}
	return payload, nil
}

func parseEnvelope(bundle []byte) (Envelope, error) {
	if len(bundle) == 0 || len(bundle) > MaxBundleSize {
		return Envelope{}, ErrInvalidBundle
	}
	var envelope Envelope
	if err := decodeStrict(bundle, &envelope); err != nil {
		return Envelope{}, ErrInvalidBundle
	}
	if envelope.Format != Format {
		return Envelope{}, ErrInvalidBundle
	}
	if envelope.Version != Version {
		return Envelope{}, ErrUnsupportedVersion
	}
	if envelope.Crypto != defaultCryptoParameters() {
		return Envelope{}, ErrInvalidBundle
	}
	if envelope.CreatedAt.IsZero() || envelope.ExpiresAt.IsZero() || !envelope.ExpiresAt.Equal(envelope.CreatedAt.Add(FileTTL)) {
		return Envelope{}, ErrInvalidBundle
	}
	if _, err := decodeExact(envelope.Salt, saltSize); err != nil {
		return Envelope{}, ErrInvalidBundle
	}
	if _, err := decodeExact(envelope.Nonce, chacha20poly1305.NonceSizeX); err != nil {
		return Envelope{}, ErrInvalidBundle
	}
	if envelope.EncryptedPayload == "" {
		return Envelope{}, ErrInvalidBundle
	}
	return envelope, nil
}

func decodeStrict(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrInvalidBundle
	}
	return nil
}

func validatePayload(payload Payload) error {
	if payload.Project == "" || payload.CreatedAt.IsZero() || payload.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: incomplete payload metadata", ErrInvalidBundle)
	}
	if !payload.ExpiresAt.Equal(payload.CreatedAt.Add(FileTTL)) {
		return fmt.Errorf("%w: file expiry must be %s", ErrInvalidBundle, FileTTL)
	}
	if len(payload.Secrets) == 0 {
		return fmt.Errorf("%w: payload contains no secrets", ErrInvalidBundle)
	}
	for _, secret := range payload.Secrets {
		if secret.Name == "" {
			return fmt.Errorf("%w: payload contains a secret without a name", ErrInvalidBundle)
		}
		if secret.Scope != store.ScopeGlobal && secret.Scope != store.ScopeProject {
			return fmt.Errorf("%w: payload contains an invalid scope", ErrInvalidBundle)
		}
		if secret.Scope == store.ScopeProject && secret.Project == "" {
			return fmt.Errorf("%w: project-scoped secret is missing its project", ErrInvalidBundle)
		}
	}
	return nil
}

func defaultCryptoParameters() CryptoParameters {
	return CryptoParameters{
		KDF:       kdfName,
		Time:      kdfTime,
		MemoryKiB: kdfMemory,
		Threads:   kdfThreads,
		Cipher:    cipherName,
	}
}

func deriveKey(passphrase, salt []byte, params CryptoParameters) []byte {
	return argon2.IDKey(passphrase, salt, params.Time, params.MemoryKiB, params.Threads, chacha20poly1305.KeySize)
}

func decodeExact(encoded string, size int) ([]byte, error) {
	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) != size {
		return nil, ErrInvalidBundle
	}
	return decoded, nil
}

func (e Envelope) additionalData() ([]byte, error) {
	type authenticatedHeader struct {
		Format    string           `json:"format"`
		Version   int              `json:"version"`
		Crypto    CryptoParameters `json:"crypto"`
		Salt      string           `json:"salt"`
		Nonce     string           `json:"nonce"`
		CreatedAt time.Time        `json:"created_at"`
		ExpiresAt time.Time        `json:"expires_at"`
	}
	return json.Marshal(authenticatedHeader{
		Format:    e.Format,
		Version:   e.Version,
		Crypto:    e.Crypto,
		Salt:      e.Salt,
		Nonce:     e.Nonce,
		CreatedAt: e.CreatedAt,
		ExpiresAt: e.ExpiresAt,
	})
}
