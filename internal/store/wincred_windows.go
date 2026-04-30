//go:build windows

package store

import (
	"context"
	"fmt"
)

// WincredStore implements Store using Windows Credential Manager.
type WincredStore struct{}

func NewWincredStore() *WincredStore {
	return &WincredStore{}
}

func (w *WincredStore) Get(_ context.Context, scope Scope, project, key string) (string, error) {
	return "", fmt.Errorf("wincred not yet implemented")
}

func (w *WincredStore) Set(_ context.Context, scope Scope, project, key, value string) error {
	return fmt.Errorf("wincred not yet implemented")
}

func (w *WincredStore) Delete(_ context.Context, scope Scope, project, key string) error {
	return fmt.Errorf("wincred not yet implemented")
}

func (w *WincredStore) List(_ context.Context, scope Scope, project string) ([]SecretEntry, error) {
	return nil, fmt.Errorf("wincred not yet implemented")
}
