//go:build darwin && !cgo

package commands

import (
	"context"
	"fmt"

	"github.com/dipockdas/keysync/internal/store"
)

// tryKeychain is not available when cgo is disabled (e.g., cross-compiling from Linux).
// The encrypted file fallback store is used instead.
func tryKeychain(_ context.Context) (store.Store, error) {
	return nil, fmt.Errorf("cgo required for macOS Keychain access")
}
