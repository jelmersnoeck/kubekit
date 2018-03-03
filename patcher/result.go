package patcher

import (
	"bytes"
	"io"

	"github.com/golang/glog"
	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

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
	mapping, err := RestMappingForObject(factory, obj)
	if err != nil {
		glog.V(4).Infof("Error getting restmapping for %s: %s", kubekit.TypeName(obj))
		return nil, err
	}

	accessor := mapping.MetadataAccessor
	namespace, err := accessor.Namespace(obj)
	if err != nil {
		glog.V(4).Infof("Error getting namespace for %s: %s", kubekit.TypeName(obj))
		return nil, err
	}

	validationScheme, err := factory.Validator(cfg.Validation)
	if err != nil {
		glog.V(4).Infof("Error getting validator for %s: %s", kubekit.TypeName(obj))
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})
	if err := factory.JSONEncoder().Encode(obj, buf); err != nil {
		glog.V(4).Infof("Error encoding the giiven object for %s: %s", kubekit.TypeName(obj))
		return nil, err
	}

	return defaultStreamBuilderFunc(factory.NewBuilder(), validationScheme, namespace, buf)
}

// RestMappingForObject returns a new RESTMApping for the given object.
func RestMappingForObject(factory Factory, obj runtime.Object) (*meta.RESTMapping, error) {
	rm, typer := factory.Object()
	gvks, _, err := typer.ObjectKinds(obj)
	if err != nil {
		return nil, err
	}
	gvk := gvks[0]
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}

	return rm.RESTMapping(gk, gvk.Version)
}
