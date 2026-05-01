//go:build linux

package commands

import (
	"context"

	"github.com/dipockdas/keysync/internal/store"
)

func tryKeychain(_ context.Context) (store.Store, error) {
	return store.NewLibsecretStore(), nil
}
