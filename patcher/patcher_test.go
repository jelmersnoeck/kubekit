package patcher_test

import (
	"testing"

	"github.com/jelmersnoeck/kubekit/errors"
	"github.com/jelmersnoeck/kubekit/patcher"
)

func TestPatcher_Apply(t *testing.T) {
	t.Run("without object to apply", func(t *testing.T) {
		p := patcher.New("test", nil)

		if _, err := p.Apply(nil); !errors.IsNoObjectGiven(err) {
			t.Errorf("Expected error to be of type `errors.ErrNoObjectGiven`, got %T", err)
		}
	})
}

func TestIsEmptyPatch(t *testing.T) {
	data := []struct {
		data []byte
		exp  bool
	}{
		{[]byte("{}"), true},
		{[]byte("{\"metadata\":{\"creationTimestamp\":null}}"), true},
		{[]byte("foo"), false},
	}

	for _, b := range data {
		if patcher.IsEmptyPatch(b.data) != b.exp {
			exp := ""
			if !b.exp {
				exp = "not "
			}
			t.Errorf("Expected '%s' %sto be EmptyPatch", string(b.data), exp)
		}
	}
}
