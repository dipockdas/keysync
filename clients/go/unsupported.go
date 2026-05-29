//go:build !darwin && !linux && !windows

package keysync

import "fmt"

func init() {
	isNotFound = func(error) bool { return true }
	platformGet = func(_, _ string) (string, error) {
		return "", fmt.Errorf("keysync client: unsupported platform (only macOS, Linux, and Windows are supported)")
	}
	platformList = func(_, _, _ string) ([]SecretEntry, error) {
		return nil, fmt.Errorf("keysync client: unsupported platform (only macOS, Linux, and Windows are supported)")
	}
}
