package cfgman

// UserInput defines the interface for user input operations
type UserInput interface {
	// ReadInput reads a line of input from the user
	ReadInput(prompt string) (string, error)
	// Confirm asks the user for a yes/no confirmation
	Confirm(prompt string) bool
	// ReadInputWithDefault reads input with a default value
	ReadInputWithDefault(prompt string, defaultValue string) (string, error)
}

// DefaultUserInput provides the default implementation of UserInput
type DefaultUserInput struct{}

// NewDefaultUserInput creates a new DefaultUserInput
func NewDefaultUserInput() UserInput {
	return &DefaultUserInput{}
}

// ReadInput reads a line of input from the user
func (d *DefaultUserInput) ReadInput(prompt string) (string, error) {
	return ReadUserInput(prompt)
}

// Confirm asks the user for a yes/no confirmation
func (d *DefaultUserInput) Confirm(prompt string) bool {
	return ConfirmPrompt(prompt)
}

// ReadInputWithDefault reads input with a default value
func (d *DefaultUserInput) ReadInputWithDefault(prompt string, defaultValue string) (string, error) {
	return ReadUserInputWithDefault(prompt, defaultValue)
}
