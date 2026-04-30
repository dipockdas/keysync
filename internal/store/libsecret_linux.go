//go:build linux

package store

import (
	"context"
	"fmt"
)

// LibsecretStore implements Store using Linux libsecret (D-Bus Secret Service).
type LibsecretStore struct{}

func NewLibsecretStore() *LibsecretStore {
	return &LibsecretStore{}
}

func (l *LibsecretStore) Get(_ context.Context, scope Scope, project, key string) (string, error) {
	return "", fmt.Errorf("libsecret not yet implemented")
}

func (l *LibsecretStore) Set(_ context.Context, scope Scope, project, key, value string) error {
	return fmt.Errorf("libsecret not yet implemented")
}

func (l *LibsecretStore) Delete(_ context.Context, scope Scope, project, key string) error {
	return fmt.Errorf("libsecret not yet implemented")
}

func (l *LibsecretStore) List(_ context.Context, scope Scope, project string) ([]SecretEntry, error) {
	return nil, fmt.Errorf("libsecret not yet implemented")
}
