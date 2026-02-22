package lnk

// Directory names to skip during status checks
const (
	// MacOS system directories
	LibraryDir = "Library"
	TrashDir   = ".Trash"
)

// File operation timeouts (in seconds)
const (
	GitCommandTimeout   = 5
	GitOperationTimeout = 10
)

// Configuration file names
const (
	ConfigFileName = ".lnkconfig" // Configuration file
	IgnoreFileName = ".lnkignore" // Gitignore-style ignore file
)

// Terminal output formatting
const (
	DryRunPrefix = "[DRY RUN]"
	SuccessIcon  = "✓"
	FailureIcon  = "✗"
	WarningIcon  = "!"
)
