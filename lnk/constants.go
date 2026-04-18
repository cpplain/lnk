package lnk

// Directory names to skip during status checks
const (
	// MacOS system directories
	LibraryDir = "Library"
	TrashDir   = ".Trash"
)

// Configuration file names
const (
	IgnoreFileName = ".lnkignore" // Gitignore-style ignore file
)

// Terminal output formatting
const (
	DryRunPrefix = "[DRY RUN]"
	SuccessIcon  = "✓"
	FailureIcon  = "✗"
	WarningIcon  = "!"
)
