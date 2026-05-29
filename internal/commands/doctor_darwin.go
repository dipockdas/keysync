//go:build darwin

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func checkMacOSSigning() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	out, err := exec.Command("codesign", "-dvvv", exe).CombinedOutput()
	s := string(out)
	if err != nil && !strings.Contains(s, "TeamIdentifier=") {
		fmt.Printf("  ⚠ macOS: could not verify code signature: %v\n", err)
		return
	}
	if strings.Contains(s, "Developer ID Application") || strings.Contains(s, "TeamIdentifier=") {
		fmt.Println("  ✓ macOS: code signed (run keysync trust after copy/rebuild)")
		return
	}
	fmt.Println("  ⚠ macOS: binary is ad-hoc or unsigned")
	fmt.Println("    Fix: make build-signed && keysync trust")
}
