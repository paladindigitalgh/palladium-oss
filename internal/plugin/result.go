package plugin

// Result is what a Plugin's Execute returns on success.
type Result struct {
	Message  string
	Metadata map[string]any
}
