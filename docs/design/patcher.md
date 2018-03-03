# Patcher

Kubekit provides a Patcher which mimics the behaviour of the `kubectl apply`
command. The key difference is that Kubekit tracks its own configuration and
leaves the kubectl configuration alone.

When using kubectl, it will annotate the last applied configuration to an object
under the key `kubectl.kubernetes.io/last-applied-configuration`. Kubekit will
do the same, but under its own key, `kubekit/last-applied-configuration`. This
is done so that we don't conflict with manual `kubectl apply` interactions.

By specifying this annotation, we can perform a three way merge patch to ensure
that we only update the fields that we should be updating specified by the CRD
change. We enforced the use of this annotation for two distinct reasons:

- controller restarts; when a controller runs and receives an update, we have
  the original state (last-applied-configuration) available together with the
  modified state. This does not survive between controller restarts when a CRD
  gets created again within the controller. Creation of objects could be ignored
  when the object already exists, but this leads us to the second point;
- three-way merge patches are not error free, especially when there is an
  interaction from an external party. By separating this from the kubectl
  annotation we 1) lower the conflict barrier; 2) allow a mechanism for further
  debugging and restoring state.
- with using an annotation and a patch, we allow other parties to edit the
  objects from outside of kubekit. This way other controllers could potentially
  interact with this as well. By specifying a name to the patcher, we ensure
  that the annotations are unique - to a certain extent - across multiple
  controllers. This allows each controller to perform its own updates
  separately.

## Terminology

We've kept the terminology of the Kubekit patcher as close as possible with the
kubectl command. For clarity, we've explained some of the core parts below.

### Original Object

The Original Object resembles the state of your object before modifying it. In
the case of a CRD, within the `OnUpdate` handler, this would resemble the
objects created from the `oldObj` version (the first argument) of the CRD.

This object could be `nil`, in case it does not yet exist. When this is the case
Kubekit will create the object in the cluster.

### Modified Object

The Modified Object resembles the state of your object which you want to
achieve. Within the `OnCreate` or `OnDelete` handlers, this is your object,
within the `OnUpdate` handler, this is the second argument, `newObj`.

This Modified Object is the object we'll store under the
`kubekit/last-applied-configuration` annotation in the cluster.

### Current Object

The Current Object resembles the state of the object as it lives on the server.
This is the full state and can hold values that are set by other interactions.
