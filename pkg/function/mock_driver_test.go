package function

// stubdriver implements inngest.Runtime, as this package cannot import
// mockdriver due to import cycles.
type stubdriver struct{}

// RuntimeType fulfiils the inngest.Runtime interface.
func (s *stubdriver) RuntimeType() string {
	return "mock"
}
