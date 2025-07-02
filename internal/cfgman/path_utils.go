package cfgman

import (
	"os"
	"path/filepath"
	"strings"
)

// ContractPath contracts the home directory to ~ in paths for display
func ContractPath(path string) string {
	if path == "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, return the original path
		return path
	}

	// Check if path starts with home directory
	if strings.HasPrefix(path, homeDir) {
		// Replace home directory with ~
		contracted := "~" + strings.TrimPrefix(path, homeDir)
		// Clean up any double slashes
		contracted = filepath.Clean(contracted)
		return contracted
	}

	return path
}
