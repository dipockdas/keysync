package keysync

import "context"

// Store defines the interface for secret storage backends.
// Implementations include OS keychain access and an in-memory store for testing.
type Store interface {
	// GetSecret retrieves a secret value. Returns ErrNotFound if not present.
	GetSecret(ctx context.Context, scope, project, key string) (string, error)
	// SetSecret stores a secret value.
	SetSecret(ctx context.Context, scope, project, key, value string) error
}
