// Package client provides an importable Go client library for reading secrets
// from the OS secret store. Applications can use this to access secrets at
// runtime without environment variables.
//
// Usage:
//
//	import "github.com/dipockdas/keysync/client"
//
//	dbURL, err := client.GetSecret("my-project", "DATABASE_URL")
package client

import (
	"context"
	"fmt"
)

// Store defines the interface for secret storage backends.
// Implementations include macOS Keychain, Linux libsecret, Windows Credential Manager,
// and an in-memory store for testing.
type Store interface {
	// GetSecret retrieves a secret value. Returns ErrNotFound if not present.
	GetSecret(ctx context.Context, scope, project, key string) (string, error)
	// SetSecret stores a secret value.
	SetSecret(ctx context.Context, scope, project, key, value string) error
}

// ErrNotFound is returned when a secret is not found.
var ErrNotFound = fmt.Errorf("secret not found")
