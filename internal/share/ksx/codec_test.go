package ksx

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dipockdas/keysync/internal/store"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	now := testTime()
	payload := testPayload(now)
	passphrase := []byte("synthetic-test-passphrase")

	bundle, err := Encrypt(payload, passphrase)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	got, err := Decrypt(bundle, passphrase, now)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got.Project != payload.Project {
		t.Fatalf("Project = %q, want %q", got.Project, payload.Project)
	}
	if len(got.Secrets) != 1 || got.Secrets[0] != payload.Secrets[0] {
		t.Fatalf("Secrets = %#v, want %#v", got.Secrets, payload.Secrets)
	}
	if !got.CreatedAt.Equal(payload.CreatedAt) || !got.ExpiresAt.Equal(payload.ExpiresAt) {
		t.Fatalf("timestamps = (%s, %s), want (%s, %s)", got.CreatedAt, got.ExpiresAt, payload.CreatedAt, payload.ExpiresAt)
	}
}

func TestEncryptUsesFreshRandomnessAndHidesSensitiveManifest(t *testing.T) {
	now := testTime()
	payload := testPayload(now)
	passphrase := []byte("synthetic-test-passphrase")

	first, err := Encrypt(payload, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encrypt(payload, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("Encrypt() produced identical bundles")
	}
	for _, sensitive := range []string{payload.Project, payload.Secrets[0].Name, payload.Secrets[0].Value} {
		if bytes.Contains(first, []byte(sensitive)) {
			t.Fatalf("bundle contains sensitive plaintext %q", sensitive)
		}
	}

	var firstEnvelope, secondEnvelope Envelope
	if err := json.Unmarshal(first, &firstEnvelope); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &secondEnvelope); err != nil {
		t.Fatal(err)
	}
	if firstEnvelope.Salt == secondEnvelope.Salt {
		t.Fatal("salt was reused")
	}
	if firstEnvelope.Nonce == secondEnvelope.Nonce {
		t.Fatal("nonce was reused")
	}
}

func TestInspectReturnsOnlyOuterMetadataAndEnforcesExpiry(t *testing.T) {
	now := testTime()
	bundle, err := Encrypt(testPayload(now), []byte("synthetic-test-passphrase"))
	if err != nil {
		t.Fatal(err)
	}

	metadata, err := Inspect(bundle, now.Add(FileTTL-time.Nanosecond))
	if err != nil {
		t.Fatalf("Inspect() before expiry error = %v", err)
	}
	if metadata.Format != Format || metadata.Version != Version {
		t.Fatalf("metadata = %#v", metadata)
	}
	if !metadata.CreatedAt.Equal(now) || !metadata.ExpiresAt.Equal(now.Add(FileTTL)) {
		t.Fatalf("metadata timestamps = %#v", metadata)
	}

	metadata, err = Inspect(bundle, now.Add(FileTTL))
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("Inspect() at expiry error = %v, want ErrExpired", err)
	}
	if !metadata.ExpiresAt.Equal(now.Add(FileTTL)) {
		t.Fatalf("expired metadata = %#v", metadata)
	}
}

func TestDecryptRejectsWrongPassphraseAndTampering(t *testing.T) {
	now := testTime()
	correctPassphrase := []byte("synthetic-test-passphrase")
	bundle, err := Encrypt(testPayload(now), correctPassphrase)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Decrypt(bundle, []byte("incorrect-test-passphrase"), now); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("wrong passphrase error = %v, want ErrAuthentication", err)
	}

	var envelope Envelope
	if err := json.Unmarshal(bundle, &envelope); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.EncryptedPayload)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)/2] ^= 1
	envelope.EncryptedPayload = base64.RawStdEncoding.EncodeToString(ciphertext)
	tamperedCiphertext, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(tamperedCiphertext, correctPassphrase, now); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered ciphertext error = %v, want ErrAuthentication", err)
	}

	if err := json.Unmarshal(bundle, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.CreatedAt = envelope.CreatedAt.Add(time.Second)
	envelope.ExpiresAt = envelope.ExpiresAt.Add(time.Second)
	tamperedMetadata, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(tamperedMetadata, correctPassphrase, now); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered metadata error = %v, want ErrAuthentication", err)
	}
}

func TestDecryptChecksExpiryBeforePassphraseAuthentication(t *testing.T) {
	now := testTime()
	bundle, err := Encrypt(testPayload(now), []byte("synthetic-test-passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Decrypt(bundle, []byte("incorrect-test-passphrase"), now.Add(FileTTL))
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("Decrypt() expired error = %v, want ErrExpired", err)
	}
}

func TestDecodeRejectsMalformedUnsupportedAndUnboundedBundles(t *testing.T) {
	now := testTime()
	passphrase := []byte("synthetic-test-passphrase")
	bundle, err := Encrypt(testPayload(now), passphrase)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		bundle []byte
		want   error
	}{
		{name: "empty", bundle: nil, want: ErrInvalidBundle},
		{name: "invalid json", bundle: []byte("not-json"), want: ErrInvalidBundle},
		{name: "trailing json", bundle: append(append([]byte(nil), bundle...), []byte(` {}`)...), want: ErrInvalidBundle},
		{name: "oversized", bundle: bytes.Repeat([]byte("x"), MaxBundleSize+1), want: ErrInvalidBundle},
		{name: "unsupported version", bundle: mutateEnvelope(t, bundle, func(envelope *Envelope) { envelope.Version++ }), want: ErrUnsupportedVersion},
		{name: "unsafe kdf memory", bundle: mutateEnvelope(t, bundle, func(envelope *Envelope) { envelope.Crypto.MemoryKiB++ }), want: ErrInvalidBundle},
		{name: "bad salt", bundle: mutateEnvelope(t, bundle, func(envelope *Envelope) { envelope.Salt = "invalid" }), want: ErrInvalidBundle},
		{name: "bad nonce", bundle: mutateEnvelope(t, bundle, func(envelope *Envelope) { envelope.Nonce = "invalid" }), want: ErrInvalidBundle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decrypt(tt.bundle, passphrase, now); !errors.Is(err, tt.want) {
				t.Fatalf("Decrypt() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestEncryptValidatesFixedTTLWithoutLeakingSecretData(t *testing.T) {
	now := testTime()
	payload := testPayload(now)
	payload.ExpiresAt = now.Add(11 * time.Minute)

	_, err := Encrypt(payload, []byte("synthetic-test-passphrase"))
	if !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("Encrypt() error = %v, want ErrInvalidBundle", err)
	}
	if strings.Contains(err.Error(), payload.Secrets[0].Name) || strings.Contains(err.Error(), payload.Secrets[0].Value) {
		t.Fatalf("error leaks secret data: %v", err)
	}
}

func TestRoundTripPreservesEmptySecretValue(t *testing.T) {
	now := testTime()
	payload := testPayload(now)
	payload.Secrets[0].Value = ""

	bundle, err := Encrypt(payload, []byte("synthetic-test-passphrase"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	got, err := Decrypt(bundle, []byte("synthetic-test-passphrase"), now)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got.Secrets[0].Value != "" {
		t.Fatalf("empty secret value changed to %q", got.Secrets[0].Value)
	}
}

func testTime() time.Time {
	return time.Date(2026, 6, 23, 9, 0, 0, 0, time.UTC)
}

func testPayload(now time.Time) Payload {
	return Payload{
		Project:   "example-app",
		CreatedAt: now,
		ExpiresAt: now.Add(FileTTL),
		Secrets: []Secret{
			{
				Name:    "SYNTHETIC_API_KEY",
				Value:   "synthetic-value-for-test-only",
				Scope:   store.ScopeProject,
				Project: "example-app",
			},
		},
	}
}

func mutateEnvelope(t *testing.T, bundle []byte, mutate func(*Envelope)) []byte {
	t.Helper()
	var envelope Envelope
	if err := json.Unmarshal(bundle, &envelope); err != nil {
		t.Fatal(err)
	}
	mutate(&envelope)
	mutated, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return mutated
}
