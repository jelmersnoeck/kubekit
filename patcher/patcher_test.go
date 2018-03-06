package patcher_test

import (
	"testing"

	"github.com/jelmersnoeck/kubekit/errors"
	"github.com/jelmersnoeck/kubekit/patcher"
)

func TestPatchResource(t *testing.T) {
	t.Run("without object to apply", func(t *testing.T) {
		p := patcher.New("test", nil)

		if err := p.Apply(nil); !errors.IsNoObjectGiven(err) {
			t.Errorf("Expected error to be of type `errors.ErrNoObjectGiven`, got %T", err)
		}
	})
}
