package lnk

// VerbosityLevel represents the output verbosity level
type VerbosityLevel int

const (
	// VerbosityQuiet suppresses all non-error output
	VerbosityQuiet VerbosityLevel = iota
	// VerbosityNormal is the default output level
	VerbosityNormal
	// VerbosityVerbose includes additional debug information
	VerbosityVerbose
)

// verbosity is the global verbosity level for the application
var verbosity = VerbosityNormal

// SetVerbosity sets the global verbosity level
func SetVerbosity(level VerbosityLevel) {
	verbosity = level
}

// GetVerbosity returns the current verbosity level
func GetVerbosity() VerbosityLevel {
	return verbosity
}

// IsQuiet returns true if running in quiet mode
func IsQuiet() bool {
	return verbosity == VerbosityQuiet
}

// IsVerbose returns true if running in verbose mode
func IsVerbose() bool {
	return verbosity == VerbosityVerbose
}
