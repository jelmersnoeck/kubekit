package kubetest

import (
	"github.com/jelmersnoeck/kubekit"

	"github.com/go-openapi/validate"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
)

// GetValidator returns a new schema validator which converts the given
// CustomResource to the correct type.
// This can be usesd to test validation rules of a created CRD.
func GetValidator(crd kubekit.CustomResource) (*validate.SchemaValidator, error) {
	val := &apiextensions.CustomResourceValidation{}
	err := v1beta1.Convert_v1beta1_CustomResourceValidation_To_apiextensions_CustomResourceValidation(
		crd.Validation,
		val,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return validation.NewSchemaValidator(val)
}
