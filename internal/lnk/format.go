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

// IsJSONFormat returns true if outputting JSON
func IsJSONFormat() bool {
	return format == FormatJSON
}
