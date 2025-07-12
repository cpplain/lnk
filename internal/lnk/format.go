package lnk

// OutputFormat represents the output format for commands
type OutputFormat int

const (
	// FormatHuman is the default human-readable format
	FormatHuman OutputFormat = iota
	// FormatJSON outputs data as JSON
	FormatJSON
)

// format is the global output format for the application
var format = FormatHuman

// SetOutputFormat sets the global output format
func SetOutputFormat(f OutputFormat) {
	format = f
}

// GetOutputFormat returns the current output format
func GetOutputFormat() OutputFormat {
	return format
}

// IsJSONFormat returns true if outputting JSON
func IsJSONFormat() bool {
	return format == FormatJSON
}
