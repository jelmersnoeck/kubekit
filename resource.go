package kubekit

import (
	"log"
	"time"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CreateCRD creates and registers a CRD with the k8s cluster.
func CreateCRD(cs clientset.Interface, c CustomResource) error {
	crd := c.Definition()

	if err := createCRD(cs, crd); err != nil {
		return err
	}

	return waitForCRD(cs, c.FullName(), crd)
}

func createCRD(cs clientset.Interface, crd *apiextv1beta1.CustomResourceDefinition) error {
	_, err := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if apierrors.IsAlreadyExists(err) {
		currentCRD, err := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// TODO(jelmer): figure out a way here to determin if we need to update
		// the object or not (do a diff between the current crd and the desired
		// crd).
		crd.ResourceVersion = currentCRD.ResourceVersion
		_, err = cs.ApiextensionsV1beta1().CustomResourceDefinitions().Update(crd)
		return err
	} else if err != nil {
		return err
	}

	return nil
}

func waitForCRD(cs clientset.Interface, fullName string, crd *apiextv1beta1.CustomResourceDefinition) error {
	err := wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Get(
			fullName,
			metav1.GetOptions{},
		)
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextv1beta1.Established:
				if cond.Status == apiextv1beta1.ConditionTrue {
					return true, err
				}
			case apiextv1beta1.NamesAccepted:
				if cond.Status == apiextv1beta1.ConditionFalse {
					log.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})

	if err != nil {
		deleteErr := cs.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(fullName, nil)
		if deleteErr != nil {
			return errors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}

	return err
}
