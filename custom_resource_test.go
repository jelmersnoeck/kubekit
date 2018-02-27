package kubekit_test

import (
	"testing"

	"github.com/jelmersnoeck/kubekit"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDefault(t *testing.T) {
	t.Run("default name", func(t *testing.T) {
		cr := kubekit.CustomResource{Object: &TestType{}}
		if name := cr.GetName(); name != "testtype" {
			t.Errorf("Expected name to be 'testtype', got '%s'", name)
		}
	})

	t.Run("default plural", func(t *testing.T) {
		cr := kubekit.CustomResource{Object: &TestType{}}
		if name := cr.GetPlural(); name != "testtypes" {
			t.Errorf("Expected name to be 'testtypes', got '%s'", name)
		}
	})

	t.Run("fullname", func(t *testing.T) {
		cr := kubekit.CustomResource{Object: &TestType{}, Group: "kubekit"}
		if name := cr.FullName(); name != "testtypes.kubekit" {
			t.Errorf("Expected name to be 'testtypes.kubekit', got '%s'", name)
		}
	})
}

func TestGroupVersionKind(t *testing.T) {
	tests := []struct {
		GVK      schema.GroupVersionKind
		Resource kubekit.CustomResource
	}{
		{
			schema.GroupVersionKind{Group: "kubekit", Version: "v1test1", Kind: "TestType"},
			kubekit.CustomResource{Group: "kubekit", Version: "v1test1", Object: &TestType{}},
		},
	}

	for _, i := range tests {
		if i.Resource.GroupVersionKind() != i.GVK {
			t.Errorf("GVK does not equal for '%s'", i.Resource.FullName())
		}
	}
}

type TestType struct{}

func (tt *TestType) DeepCopyObject() runtime.Object  { return tt }
func (_ *TestType) GetObjectKind() schema.ObjectKind { return nil }

func TestTypeName(t *testing.T) {
	tests := []struct {
		Name   string
		Object interface{}
	}{
		{"TestType", &TestType{}},
		{"TestType", TestType{}},
		{"CustomResource", &kubekit.CustomResource{}},
		{"CustomResource", kubekit.CustomResource{}},
	}

	for _, i := range tests {
		if name := kubekit.TypeName(i.Object); name != i.Name {
			t.Errorf("Expected name to be '%s', got '%s'", i.Name, name)
		}
	}

}
