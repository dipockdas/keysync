//go:build linux

package commands

import (
	"context"
	"fmt"

	"github.com/dipockdas/keysync/internal/store"
)

func tryKeychain(_ context.Context) (store.Store, error) {
	return nil, fmt.Errorf("no keychain on linux")
}
