package lnk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// removeFromRepository removes a file from the repository (both git tracking and filesystem)
func removeFromRepository(path string) error {
	// Try git rm first - it will handle both git tracking and filesystem removal
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rm", "-f", "--", path)
	cmd.Dir = filepath.Dir(path)

	if output, err := cmd.CombinedOutput(); err == nil {
		// Success! File removed from both git and filesystem
		return nil
	} else if ctx.Err() == context.DeadlineExceeded {
		// Command timed out
		PrintVerbose("git rm timed out, falling back to filesystem removal")
	} else if len(output) > 0 {
		// Log git output but don't fail
		PrintVerbose("git rm failed: %s", strings.TrimSpace(string(output)))
	}

	// Git rm failed (not in git repo, file not tracked, git not available, etc.)
	// Just remove from filesystem
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}

	return nil
}
