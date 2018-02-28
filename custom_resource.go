package kubekit

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CustomResource describes the configuration values for a
// CustomResourceDefinition.
type CustomResource struct {
	Name    string
	Plural  string
	Group   string
	Version string
	Aliases []string
	Scope   v1beta1.ResourceScope
	Object  runtime.Object
}

// GetName returns the CustomResource Name. If no name is specified, the
// lowercased kind is used.
func (c CustomResource) GetName() string {
	if c.Name != "" {
		return c.Name
	}

	return strings.ToLower(c.Kind())
}

// GetPlural returns the pluralisation of the CustomResource. If no Plural value
// is specified, an 's' will be added to the Name.
func (c CustomResource) GetPlural() string {
	if c.Plural != "" {
		return c.Plural
	}

	return fmt.Sprintf("%ss", c.GetName())
}

// FullName returns the combined name of the Plural and the Group.
func (c CustomResource) FullName() string {
	return c.GetPlural() + "." + c.Group
}

// GroupVersion returns the GroupVersion Schema representation of this
// CustomResource.
func (c CustomResource) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{
		Group:   c.Group,
		Version: c.Version,
	}
}

// GroupVersionKind returns the GroupVersionKind Schema representation of this
// CustomResource.
func (c CustomResource) GroupVersionKind() schema.GroupVersionKind {
	return c.GroupVersion().WithKind(c.Kind())
}

// Kind returns the Type Name of the CustomResource Object.
func (c CustomResource) Kind() string {
	return TypeName(c.Object)
}

// Definition returns the CustomResourceDefinition that is linked to this
// CustomResource.
func (c CustomResource) Definition() *apiextv1beta1.CustomResourceDefinition {
	return &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.FullName(),
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   c.Group,
			Version: c.Version,
			Scope:   c.Scope,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural:     c.Plural,
				ShortNames: c.Aliases,
				Kind:       c.Kind(),
			},
		},
	}
}

// TypeName returns the Type Name of a given object.
func TypeName(o interface{}) string {
	val := reflect.ValueOf(o)

	var name string
	switch val.Kind() {
	case reflect.Ptr:
		name = val.Elem().Type().Name()
	default:
		name = val.Type().Name()
	}

	return name
}
