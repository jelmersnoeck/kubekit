package errors

import "errors"

var (
	// ErrCreateNotAllowed is used for when AllowCreate is disabled and a create
	// action is performed.
	ErrCreateNotAllowed = errors.New("Creating an object is not allowed with the current configuration")

	// ErrUpdateNotAllowed is used for when AllowUpdate is disabled and a update
	// action is performed.
	ErrUpdateNotAllowed = errors.New("Updating an object is not allowed with the current configuration")

	// ErrNoObjectGiven is used when apply is called with a nil object.
	ErrNoObjectGiven = errors.New("`nil` object given, can't apply")

	// ErrNoRuntimeObject is used when a passed through object interface doesn't
	// match the runtime.Object interface.
	ErrNoRuntimeObject = errors.New("Object is not of type runtime.Object")

	// ErrNoPointerObject is used when the passed object is not a pointer.
	ErrNoPointerObject = errors.New("Given object is not a pointer")
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

// IsNoObjectGiven will return wether or not the provided error equals
// ErrNoObjectGiven.
func IsNoObjectGiven(err error) bool {
	return errEquals(ErrNoObjectGiven, err)
}

func errEquals(expected, actual error) bool {
	return expected == actual
}
