package patcher

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

// GetOriginalConfiguration retrieves the original configuration of the object
// from the annotation, or nil if no annotation was found.
func GetOriginalConfiguration(name string, mapping *meta.RESTMapping, obj runtime.Object) ([]byte, error) {
	annots, err := mapping.MetadataAccessor.Annotations(obj)
	if err != nil {
		return nil, err
	}

	if annots == nil {
		return nil, nil
	}

	original, ok := annots[namespacedAnnotation(name)]
	if !ok {
		return nil, nil
	}

	return []byte(original), nil
}

// SetOriginalConfiguration sets the original configuration of the object
// as the annotation on the object for later use in computing a three way patch.
func SetOriginalConfiguration(name string, info *resource.Info, original []byte) error {
	if len(original) < 1 {
		return nil
	}

	accessor := info.Mapping.MetadataAccessor
	annots, err := accessor.Annotations(info.Object)
	if err != nil {
		return err
	}

	if annots == nil {
		annots = map[string]string{}
	}

	annots[namespacedAnnotation(name)] = string(original)
	return info.Mapping.MetadataAccessor.SetAnnotations(info.Object, annots)
}

// GetModifiedConfiguration retrieves the modified configuration of the object.
// If annotate is true, it embeds the result as an annotation in the modified
// configuration. If an object was read from the command input, it will use that
// version of the object. Otherwise, it will use the version from the server.
func GetModifiedConfiguration(name string, info *resource.Info, annotate bool, codec runtime.Encoder) ([]byte, error) {
	// First serialize the object without the annotation to prevent recursion,
	// then add that serialization to it as the annotation and serialize it again.
	var modified []byte

	// Otherwise, use the server side version of the object.
	accessor := info.Mapping.MetadataAccessor
	// Get the current annotations from the object.
	annots, err := accessor.Annotations(info.Object)
	if err != nil {
		return nil, err
	}

	if annots == nil {
		annots = map[string]string{}
	}

	original := annots[namespacedAnnotation(name)]
	delete(annots, namespacedAnnotation(name))
	if err := accessor.SetAnnotations(info.Object, annots); err != nil {
		return nil, err
	}

	modified, err = runtime.Encode(codec, info.Object)
	if err != nil {
		return nil, err
	}

	if annotate {
		annots[namespacedAnnotation(name)] = string(modified)
		if err := info.Mapping.MetadataAccessor.SetAnnotations(info.Object, annots); err != nil {
			return nil, err
		}

		modified, err = runtime.Encode(codec, info.Object)
		if err != nil {
			return nil, err
		}
	}

	// Restore the object to its original condition.
	annots[namespacedAnnotation(name)] = original
	if err := info.Mapping.MetadataAccessor.SetAnnotations(info.Object, annots); err != nil {
		return nil, err
	}

	return modified, nil
}

// CreateApplyAnnotation gets the modified configuration of the object,
// without embedding it again, and then sets it on the object as the annotation.
func CreateApplyAnnotation(name string, info *resource.Info, codec runtime.Encoder) error {
	modified, err := GetModifiedConfiguration(name, info, false, codec)
	if err != nil {
		return err
	}
	return SetOriginalConfiguration(name, info, modified)
}

func namespacedAnnotation(name string) string {
	return fmt.Sprintf("kubekit-%s/last-applied-configuration", name)
}
