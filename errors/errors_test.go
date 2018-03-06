package errors_test

import (
	"fmt"
	"testing"

	"github.com/jelmersnoeck/kubekit/errors"
)

func TestErrors(t *testing.T) {
	errs := []struct {
		check func(error) bool
		err   error
	}{
		{errors.IsCreateNotAllowed, errors.ErrCreateNotAllowed},
		{errors.IsUpdateNotAllowed, errors.ErrUpdateNotAllowed},
		{errors.IsNoObjectGiven, errors.ErrNoObjectGiven},
	}

	for _, err := range errs {
		if !err.check(err.err) {
			t.Errorf("Error %T does not match expected error", err.err)
		}

		if err.check(fmt.Errorf("bad error: %s", err.err.Error())) {
			t.Errorf("Error %T does matches wrong error type", err.err)
		}
	}
}
