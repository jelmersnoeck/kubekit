package patcher

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/golang/glog"
	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

var defaultNamespace = "default"

// builderFunc is used to define the function we'll use to build our results.
// This is represented as an interface here so we can overwrite it in tests and
// create our own results.
type builderFunc func(b *resource.Builder, validationScheme validation.Schema, namespace string, stream io.Reader) (Result, error)

var defaultStreamBuilderFunc builderFunc = func(b *resource.Builder, validationScheme validation.Schema, namespace string, stream io.Reader) (Result, error) {
	return b.Unstructured().
		Schema(validationScheme).
		ContinueOnError().
		NamespaceParam(namespace).
		DefaultNamespace().
		Stream(stream, "").
		Flatten().
		Do().Infos()
}

// Result provides convenience methods for comparing collections of Infos.
type Result []*resource.Info

// Visit implements resource.Visitor.
func (r Result) Visit(fn resource.VisitorFunc) error {
	for _, i := range r {
		if err := fn(i, nil); err != nil {
			return err
		}
	}

	return nil
}

// NewResult creats a new Result set based on the givven mapping and
// configuration.
func NewResult(cfg *Config, factory Factory, obj runtime.Object) (Result, error) {
	validationScheme, err := factory.Validator(cfg.Validation)
	if err != nil {
		glog.V(4).Infof("Error getting validator for %s: %s", kubekit.TypeName(obj), err)
		return nil, err
	}

	jsonData, err := json.Marshal(obj)
	if err != nil {
		glog.V(4).Infof("Error encoding the given object for %s: %s", kubekit.TypeName(obj), err)
		return nil, err
	}

	return defaultStreamBuilderFunc(
		factory.NewBuilder(),
		validationScheme,
		defaultNamespace,
		bytes.NewBuffer(jsonData),
	)
}
