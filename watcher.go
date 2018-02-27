package kubekit

import (
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// ResyncPeriod is the delay between resync actions from the controller. This
// can be overwritten at package level to define the ResyncPeriod for the
// controller.
var ResyncPeriod = 5 * time.Second

// Watcher represents a CRD Watcher Object. It knows enough details about a CRD
// to be able to create a controller and watch for changes.
type Watcher struct {
	rc        *rest.RESTClient
	namespace string
	resource  *CustomResource
	handler   cache.ResourceEventHandlerFuncs
}

// NewWatcher returns a new watcher that can be used to watch in a given
// namespace. If namespace is an empty string, all namespaces will be watched.
func NewWatcher(rc *rest.RESTClient, namespace string, resource *CustomResource, handler cache.ResourceEventHandlerFuncs) *Watcher {
	return &Watcher{
		rc:        rc,
		namespace: namespace,
		resource:  resource,
		handler:   handler,
	}
}

// Run starts watching the CRDs associated with the Watcher through a
// Kubernetes CacheController.
func (w *Watcher) Run(done <-chan struct{}) {
	source := cache.NewListWatchFromClient(
		w.rc,
		w.resource.Plural,
		w.namespace,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		source,
		w.resource.Object,
		ResyncPeriod,
		w.handler,
	)

	go controller.Run(done)
}
