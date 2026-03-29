package unifi

import "fmt"

// ErrAuth indicates an authentication failure (401).
type ErrAuth struct {
	Message string
	Body    string
}

func (e *ErrAuth) Error() string { return fmt.Sprintf("auth error: %s", e.Message) }

// ErrNotFound indicates the requested resource was not found (404).
type ErrNotFound struct {
	Message string
	Body    string
}

func (e *ErrNotFound) Error() string { return fmt.Sprintf("not found: %s", e.Message) }

// ErrRateLimit indicates the API rate limit was exceeded (429).
type ErrRateLimit struct {
	Message string
	Body    string
}

func (e *ErrRateLimit) Error() string { return fmt.Sprintf("rate limit: %s", e.Message) }

// ErrConnection indicates a network connectivity failure.
type ErrConnection struct {
	Err error
}

func (e *ErrConnection) Error() string { return fmt.Sprintf("connection error: %v", e.Err) }
func (e *ErrConnection) Unwrap() error { return e.Err }

// ErrVersionIncompatible indicates the controller is missing required capabilities.
type ErrVersionIncompatible struct {
	Message string
	Missing []string // endpoint paths or feature flags that are missing
}

func (e *ErrVersionIncompatible) Error() string {
	return fmt.Sprintf("version incompatible: %s", e.Message)
}

// ErrAPI is a generic API error for unmapped status codes.
type ErrAPI struct {
	StatusCode int
	Body       string
}

func (e *ErrAPI) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an ErrNotFound.
func IsNotFound(err error) bool {
	_, ok := err.(*ErrNotFound)
	return ok
}

// IsAuth reports whether err is an ErrAuth.
func IsAuth(err error) bool {
	_, ok := err.(*ErrAuth)
	return ok
}
