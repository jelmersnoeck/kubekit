// Package patcher is heavily inspired by the work done in the Kubernetes
// kubectl package (https://github.com/kubernetes/kubernetes/tree/master/pkg/kubectl)
// It has been altered to make it easier to use from an extension point of view.
package patcher

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/jelmersnoeck/kubekit"
	kerrors "github.com/jelmersnoeck/kubekit/errors"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubectl"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

var backoffPeriod = time.Second

// Factory represents a slimmed down version of the kubectl cmdutil
// Factory. It is recommended to use this factory to inject into the patcher,
// but you can provide your own implementation as well.
type Factory interface {
	Object() (meta.RESTMapper, runtime.ObjectTyper)
	Validator(validate bool) (validation.Schema, error)
	NewBuilder() *resource.Builder
	ClientSet() (internalclientset.Interface, error)
	JSONEncoder() runtime.Encoder
	Decoder(bool) runtime.Decoder
	OpenAPISchema() (openapi.Resources, error)
}

// Patcher represents the PatcherObject which is responsible for applying
// changes to a resource upstream.
type Patcher struct {
	Factory
	cfg *Config
}

// New sets up a new Patcher which can perform a SimpleApply and an Apply. The
// optional options provided will act as defaults for these operations. Any
// options given in the specific methods will take president over the defaults.
func New(name string, f Factory, opts ...OptionFunc) *Patcher {
	opts = append(opts, withName(name))
	return &Patcher{Factory: f, cfg: NewConfig(opts...)}
}

// Apply will take an object and calculate a patch for it to apply to the
// server. When an object hasn't been created before, the object will be created
// unless otherwise specified.
// By using apply, Kubekit will annotate the resource on the server to keep
// track of applied changes so it can perform a three-way merge.
func (p *Patcher) Apply(obj runtime.Object, opts ...OptionFunc) ([]byte, error) {
	if obj == nil {
		return nil, kerrors.ErrNoObjectGiven
	}

	cfg := NewFromConfig(p.cfg, opts...)

	r, err := NewResult(cfg, p.Factory, obj)
	if err != nil {
		return nil, err
	}

	os, err := p.OpenAPISchema()
	if err != nil {
		return nil, err
	}

	encoder := p.JSONEncoder()
	var patch []byte
	err = r.Visit(func(info *resource.Info, err error) error {
		// Get the modified configuration of the object.
		modified, err := GetModifiedConfiguration(p.cfg.name, info, true, encoder)
		if err != nil {
			kubekit.Logger.Infof("Error getting the modified configuration for %s: %s", info.Name, err)
			return err
		}
		patch = modified

		// Load the current object that is available on the server into our Info
		// object.
		if err := info.Get(); err != nil {
			if !errors.IsNotFound(err) {
				kubekit.Logger.Infof("Error getting the server object for %s: %s", info.Name, err)
				return err
			}

			if cfg.AllowCreate {
				// Apply annotations to the object so we can track future changes.
				if err := CreateApplyAnnotation(p.cfg.name, info, encoder); err != nil {
					kubekit.Logger.Infof("Error creating apply annotations for %s: %s", info.Name, err)
					return err
				}
				if err := createAndRefresh(info); err != nil {
					kubekit.Logger.Infof("Error creating the resource for %s: %s", info.Name, err)
					return err
				}

				if _, err := info.Mapping.UID(info.Object); err != nil {
					kubekit.Logger.Infof("Error getting a UID for %s: %s", info.Name, err)
					return err
				}

				return nil
			}

			return kerrors.ErrCreateNotAllowed
		}

		if cfg.AllowUpdate {
			op := &objectPatcher{
				cfg:           cfg,
				namespace:     info.Namespace,
				name:          info.Name,
				mapping:       info.Mapping,
				helper:        newHelper(info),
				encoder:       encoder,
				decoder:       p.Decoder(false),
				clientsetFunc: p.Factory.ClientSet,
				openapiSchema: os,
			}

			patch, err = op.patch(info.Object, modified)
			return err
		}

		return kerrors.ErrUpdateNotAllowed
	})

	return patch, err
}

// Delete will delete the object from the cluster. It will try and do it
// gracefully, if that's not possible, it will force deletion.
func (p *Patcher) Delete(obj runtime.Object, opts ...OptionFunc) error {
	if obj == nil {
		return kerrors.ErrNoObjectGiven
	}

	cfg := NewFromConfig(p.cfg, opts...)

	r, err := NewResult(cfg, p.Factory, obj)
	if err != nil {
		return err
	}

	return r.Visit(func(info *resource.Info, err error) error {
		op := &objectPatcher{
			cfg:           cfg,
			namespace:     info.Namespace,
			name:          info.Name,
			mapping:       info.Mapping,
			helper:        newHelper(info),
			clientsetFunc: p.Factory.ClientSet,
		}

		return op.delete()
	})
}

// Get fetches the data for a given object in a given namespace with the given
// name and loads it into the given object.
func (p *Patcher) Get(obj interface{}, namespace, name string) error {
	if obj == nil {
		return kerrors.ErrNoObjectGiven
	}

	robj, ok := obj.(runtime.Object)
	if !ok {
		return kerrors.ErrNoRuntimeObject
	}

	if reflect.ValueOf(obj).Kind() != reflect.Ptr {
		return kerrors.ErrNoPointerObject
	}

	helper, err := p.helper(robj)
	if err != nil {
		return err
	}

	nobj, err := helper.Get(namespace, name, false)
	if err != nil {
		return err
	}

	rawData, err := json.Marshal(nobj)
	if err != nil {
		return err
	}

	return json.Unmarshal(rawData, obj)
}

func (p *Patcher) helper(obj runtime.Object) (*resource.Helper, error) {
	cfg := NewFromConfig(p.cfg)

	r, err := NewResult(cfg, p.Factory, obj)
	if err != nil {
		return nil, err
	}

	var helper *resource.Helper
	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		helper = newHelper(info)
		return nil
	})

	return helper, err
}

// IsEmptyPatch looks at the contents of a patch to see wether or not it is an
// empty patch and could thus potentially be skipped.
// JSONMergePatch doesn't always cleanly merge, so we need to set up a set of
// rules we can ignore.
func IsEmptyPatch(patch []byte) bool {
	emptySets := []string{
		"{\"metadata\":{\"creationTimestamp\":null}}",
		"{}",
		"{\"metadata\":{\"annotations\":{}}}",
		"{\"metadata\":{\"labels\":{}}}",
	}
	patchString := string(patch)
	for _, s := range emptySets {
		if patchString == s {
			return true
		}
	}

	return false
}

type objectPatcher struct {
	encoder runtime.Encoder
	decoder runtime.Decoder

	namespace string
	name      string

	cfg *Config

	mapping       *meta.RESTMapping
	helper        *resource.Helper
	clientsetFunc func() (internalclientset.Interface, error)
	openapiSchema openapi.Resources
}

func (p *objectPatcher) patchSimple(obj runtime.Object, modified []byte) ([]byte, error) {
	// Load the original configuration from the annotation that we've set up
	// in the object that is currently on the server.
	original, err := GetOriginalConfiguration(p.cfg.name, p.mapping, obj)
	if err != nil {
		kubekit.Logger.Infof("Error getting the original configuration for %s: %s", p.name, err)
		return nil, err
	}

	// Load the current object as a JSON structure from the Object we've
	// previously loaded from the server.
	current, err := runtime.Encode(p.encoder, obj)
	if err != nil {
		kubekit.Logger.Infof("Error encoding the current object for %s: %s", p.name, err)
		return nil, err
	}

	var patchType types.PatchType
	var patch []byte

	versionedObject, err := scheme.Scheme.New(p.mapping.GroupVersionKind)

	// CRDs in k8s 1.9+ count as being registered, and so will not have errored.
	// They are considered unstructured and can't be strategic merge patched,
	// however. Check for this.
	_, isUnstructured := versionedObject.(runtime.Unstructured)

	switch {
	case runtime.IsNotRegisteredError(err), isUnstructured:
		patchType = types.MergePatchType
		preconditions := []mergepatch.PreconditionFunc{
			mergepatch.RequireKeyUnchanged("apiVersion"),
			mergepatch.RequireKeyUnchanged("kind"),
			mergepatch.RequireMetadataKeyUnchanged("name"),
		}
		patch, err = jsonmergepatch.CreateThreeWayJSONMergePatch(
			original,
			modified,
			current,
			preconditions...,
		)
		if err != nil {
			if mergepatch.IsPreconditionFailed(err) {
				return nil, fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
			}

			return nil, err
		}
	case err != nil:
		return nil, err
	case err == nil:
		patchType = types.StrategicMergePatchType
		patch, err = strategicMergePatch(p.openapiSchema, p.mapping.GroupVersionKind, versionedObject, original, modified, current)
		if err != nil {
			return nil, err
		}
	}

	if IsEmptyPatch(patch) {
		return patch, nil
	}

	_, err = p.helper.Patch(p.namespace, p.name, patchType, patch)
	return patch, err
}

func (p *objectPatcher) patch(current runtime.Object, modified []byte) ([]byte, error) {
	patch, err := p.patchSimple(current, modified)
	var getErr error
	for i := 1; i <= p.cfg.Retries && errors.IsConflict(err); i++ {
		// perform exponential backoff.
		time.Sleep(time.Duration(int32(i)) * backoffPeriod)

		// object could have been updated in the meantime due to exponential
		// backoff, refresh.
		current, getErr = p.helper.Get(p.namespace, p.name, false)
		if getErr != nil {
			return nil, err
		}

		patch, err = p.patchSimple(current, modified)
	}

	if err != nil && errors.IsConflict(err) && p.cfg.Force {
		patch, err = p.deleteAndCreate(current, modified)
	}

	if err != nil && !IsEmptyPatch(patch) && p.cfg.DeleteFirst {
		patch, err = p.deleteAndCreate(current, modified)
	}

	return patch, err
}

func (p *objectPatcher) deleteAndCreate(original runtime.Object, modified []byte) ([]byte, error) {
	if err := p.delete(); err != nil {
		return modified, err
	}

	err := wait.PollImmediate(kubectl.Interval, 0, func() (bool, error) {
		if _, err := p.helper.Get(p.namespace, p.name, false); !errors.IsNotFound(err) {
			return false, err
		}
		return true, nil
	})

	if err != nil {
		return modified, err
	}

	return p.create(modified)
}

func (p *objectPatcher) create(modified []byte) ([]byte, error) {
	versionedObject, _, err := p.decoder.Decode(modified, nil, nil)
	if err != nil {
		return modified, err
	}

	_, err = p.helper.Create(p.namespace, true, versionedObject)
	return modified, err
}

func (p *objectPatcher) delete() error {
	cs, err := p.clientsetFunc()
	if err != nil {
		return err
	}

	// look for an available reaper to gracefully shut down the underlying
	// objects. If none exists, force delete.
	r, err := kubectl.ReaperFor(p.mapping.GroupVersionKind.GroupKind(), cs)
	if err != nil {
		if _, ok := err.(*kubectl.NoSuchReaperError); !ok {
			return err
		}

		return p.helper.Delete(p.namespace, p.name)
	}

	return r.Stop(p.namespace, p.name, 2*time.Minute, nil)
}

// createAndRefresh creates an object from input info and refreshes info with that object
func createAndRefresh(info *resource.Info) error {
	obj, err := newHelper(info).Create(info.Namespace, true, info.Object)
	if err != nil {
		log.Printf("Error using helper")
		return err
	}
	info.Refresh(obj, true)
	return nil
}

func newHelper(info *resource.Info) *resource.Helper {
	return resource.NewHelper(info.Client, info.Mapping)
}

func strategicMergePatch(schema openapi.Resources, gvk schema.GroupVersionKind, obj runtime.Object, original, modified, current []byte) ([]byte, error) {
	patch, err := openapiPatch(schema, gvk, original, modified, current)

	// no need to return the error, we'll try a regular patch if this fails
	if err != nil {
		log.Printf("warning: error calculating patch from openapi spec: %v\n", err)
	}

	if patch != nil {
		return patch, nil
	}

	return strategicPatch(obj, original, modified, current)
}

func openapiPatch(schema openapi.Resources, gvk schema.GroupVersionKind, original, modified, current []byte) ([]byte, error) {
	if schema == nil {
		return nil, nil
	}

	os := schema.LookupResource(gvk)
	if os == nil {
		return nil, nil
	}

	lookupPatchMeta := strategicpatch.PatchMetaFromOpenAPI{Schema: os}
	return threeWayMergePatch(original, modified, current, lookupPatchMeta)
}

func strategicPatch(obj runtime.Object, original, modified, current []byte) ([]byte, error) {
	lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(obj)
	if err != nil {
		return nil, err
	}

	return threeWayMergePatch(original, modified, current, lookupPatchMeta)
}

func threeWayMergePatch(original, modified, current []byte, meta strategicpatch.LookupPatchMeta) ([]byte, error) {
	return strategicpatch.CreateThreeWayMergePatch(original, modified, current, meta, true)
}
