package errors

import "errors"

var (
	// ErrCreateNotAllowed is used for when AllowCreate is disabled and a create
	// action is performed.
	ErrCreateNotAllowed = errors.New("Creating an object is not allowed with the current configuration")

	// ErrUpdateNotAllowed is used for when AllowUpdate is disabled and a update
	// action is performed.
	ErrUpdateNotAllowed = errors.New("Updating an object is not allowed with the current configuration")
)

// IsCreateNotAllowed will return wether or not the provided error equals
// ErrCreateNotAllowed.
func IsCreateNotAllowed(err error) bool {
	return errEquals(ErrCreateNotAllowed, err)
}

// IsUpdateNotAllowed will return wether or not the provided error equals
// ErrUpdateNotAllowed.
func IsUpdateNotAllowed(err error) bool {
	return errEquals(ErrUpdateNotAllowed, err)
}

func errEquals(expected, actual error) bool {
	return expected == actual
}
