package kubekit

import (
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Clientsets returns a set of clientsets for the given configuration.
func Clientsets(cfg *rest.Config) (kubernetes.Interface, clientset.Interface, error) {
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	ac, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return kc, ac, nil
}

// InClusterClientsets creates the in cluster configured clientsets with the
// rest.InClusterConfig.
func InClusterClientsets() (*rest.Config, kubernetes.Interface, clientset.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	kc, ac, err := Clientsets(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	return cfg, kc, ac, nil
}

// SchemeBuilder allows us to add runtime.Scheme objects to our RESTClient.
type SchemeBuilder func(*runtime.Scheme) error

// RESTClient configures a new REST Client to be able to understand all the
// schemes defined. This way users can query objects associated with this
// scheme.
func RESTClient(cfg *rest.Config, sgv *schema.GroupVersion, schemeBuilders ...SchemeBuilder) (*rest.RESTClient, error) {
	scheme := runtime.NewScheme()

	for _, builder := range schemeBuilders {
		if err := builder(scheme); err != nil {
			return nil, err
		}
	}

	config := *cfg
	config.GroupVersion = sgv
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme),
	}

	return rest.RESTClientFor(&config)
}
