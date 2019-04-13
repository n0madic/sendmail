package sendmail

// Level type of result
type Level uint32

const (
	// FatalLevel level.
	FatalLevel Level = iota
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the application.
	InfoLevel
)

// Fields type, used for expand information.
type Fields map[string]interface{}

// Result of send
type Result struct {
	Level   Level
	Error   error
	Message string
	Fields  Fields
}
