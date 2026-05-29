package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if pub == nil {
		t.Fatal("public key is nil")
	}
	if priv == nil {
		t.Fatal("secret key is nil")
	}
}

func TestGenerateRandomKey(t *testing.T) {
	key, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("GenerateRandomKey failed: %v", err)
	}
	if key == nil {
		t.Fatal("key is nil")
	}
	// Verify length
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
}

func TestKeyToBytes(t *testing.T) {
	key := &[32]byte{}
	for i := range key {
		key[i] = byte(i)
	}
	b := KeyToBytes(key)
	if len(b) != 32 {
		t.Errorf("len = %d, want 32", len(b))
	}
	for i, v := range b {
		if v != byte(i) {
			t.Errorf("b[%d] = %d, want %d", i, v, i)
		}
	}
}

func TestBytesToKey(t *testing.T) {
	input := make([]byte, 32)
	for i := range input {
		input[i] = byte(i)
	}
	key, err := BytesToKey(input)
	if err != nil {
		t.Fatalf("BytesToKey failed: %v", err)
	}
	for i, v := range key {
		if v != byte(i) {
			t.Errorf("key[%d] = %d, want %d", i, v, i)
		}
	}
}

func TestBytesToKey_WrongLength(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", make([]byte, 16)},
		{"too long", make([]byte, 64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BytesToKey(tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestNewSealedBox(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	sb := NewSealedBox(pub, priv)
	if sb == nil {
		t.Fatal("NewSealedBox returned nil")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	pub, priv, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	sb := NewSealedBox(pub, priv)

	plaintext := []byte("hello, this is a secret message")
	ciphertext, err := sb.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := sb.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("decrypted = %q, want %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	pub, priv, _ := GenerateKey()
	sb := NewSealedBox(pub, priv)

	ciphertext, err := sb.Encrypt([]byte{})
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err := sb.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("decrypted len = %d, want 0", len(decrypted))
	}
}

func TestEncryptDecrypt_LargePlaintext(t *testing.T) {
	pub, priv, _ := GenerateKey()
	sb := NewSealedBox(pub, priv)

	plaintext := make([]byte, 1_000_000)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ciphertext, err := sb.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt large failed: %v", err)
	}

	decrypted, err := sb.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt large failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("large plaintext round-trip mismatch")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	pub, priv, _ := GenerateKey()
	sb := NewSealedBox(pub, priv)

	plaintext := []byte("secret data")
	ciphertext, err := sb.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Create a box with different key pair
	pub2, priv2, _ := GenerateKey()
	sb2 := NewSealedBox(pub2, priv2)

	_, err = sb2.Decrypt(ciphertext)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key, got nil")
	}
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	pub, priv, _ := GenerateKey()
	sb := NewSealedBox(pub, priv)

	ciphertext, err := sb.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Corrupt one byte
	ciphertext[len(ciphertext)/2] ^= 0xFF

	_, err = sb.Decrypt(ciphertext)
	if err == nil {
		t.Fatal("expected error decrypting corrupted data, got nil")
	}
}

func TestGenerateRandomKey_Unique(t *testing.T) {
	k1, _ := GenerateRandomKey()
	k2, _ := GenerateRandomKey()
	if *k1 == *k2 {
		t.Error("two random keys should not be equal")
	}
}
