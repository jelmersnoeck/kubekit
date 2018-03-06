package patcher

import "errors"

// Config represents a set of options that can be passed into an Apply action.
type Config struct {
	// AllowCreate specifies wether or not we should be able to create the
	// object or not. If this is disabled, when an object does not exist on the
	// server and a patch is requested, Kubekit will return an error.
	// Disabling this could be useful to ensure the `OnUpdate` CRD Handler
	// function only performs updates, not creates.
	// This does not count towards the `Force` option, where we'll delete and
	// re-create an object if there is an error on updating it.
	// Defaults to `true`
	AllowCreate bool

	// AllowUpdate specifies wether or not we should be able to perform updates.
	// If this is disabled and, when an object already exists on the server and
	// a patch is sent, Kubekit will return an error.
	// Disabling this could be useful to ensure the `OnCreate` CRD Handler
	// function only performs create actions.
	// Defaults to `true`
	AllowUpdate bool

	// DeleteFirst enforces us to delete the resource on the server first before
	// trying to patch it. This enforces creating a new resource.
	// This option is provided to enable replacing specific resources like
	// PodDisruptionBudget. These resources can't be updated and need to be
	// recreated to reconfigure.
	// Defaults to `false`
	DeleteFirst bool

	// Force allows Kubekit to delete and re-create the object when there is an
	// error applying the patch.
	// This can come in handy for objects that don't allow updating, like
	// PodDisruptionBudget.
	// Defaults to `false`
	Force bool

	// Validation enables the schema validation before sending it off to the
	// server.
	// Defaults to `false`
	Validation bool

	// Retries resembles the amount of retries we'll execute when we encounter
	// an error applying a patch.
	// Defaults to `5`
	Retries int

	name string
}

var defaultOptions = &Config{
	AllowCreate: true,
	AllowUpdate: true,
	Force:       false,
	Validation:  true,
	Retries:     5,
}

var (
	// ErrCreateNotAllowed is used for when AllowCreate is disabled and a create
	// action is performed.
	ErrCreateNotAllowed = errors.New("Creating an object is not allowed with the current configuration")

	// ErrUpdateNotAllowed is used for when AllowUpdate is disabled and a update
	// action is performed.
	ErrUpdateNotAllowed = errors.New("Updating an object is not allowed with the current configuration")
)

// OptionFunc represents a function that can be used to set options for the
// Apply command.
type OptionFunc func(c *Config)

// NewConfig creates a new configuration. Any options passed in will overwrite
// the defaults.
func NewConfig(opts ...OptionFunc) *Config {
	return NewFromConfig(defaultOptions, opts...)
}

// NewFromConfig creates a new configuration based off of the given
// configuration and options. The given options will overwrite the specified
// config.
func NewFromConfig(c *Config, opts ...OptionFunc) *Config {
	cfg := c.DeepCopy()

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// DeepCopy copies the entire config object to a new struct.
func (c *Config) DeepCopy() *Config {
	cfg := *c
	return &cfg
}

// DisableValidation disables the schema validation.
func DisableValidation() OptionFunc {
	return func(c *Config) {
		c.Validation = false
	}
}

// DisableCreate disables creating objects. They can only be updated.
func DisableCreate() OptionFunc {
	return func(c *Config) {
		c.AllowCreate = false
	}
}

// DisableUpdate disables updating objects. They can only be created.
func DisableUpdate() OptionFunc {
	return func(c *Config) {
		c.AllowUpdate = false
	}
}

// WithForce Delete and re-create the specified object when there is an error
// and we've retried several times.
// This can come in handy for objects that don't allow updating, like
// PodDisruptionBudget.
func WithForce() OptionFunc {
	return func(c *Config) {
		c.Force = true
	}
}

// WithRetries sets the amount of retries we should execute when encountering
// an error before backing off.
func WithRetries(i int) OptionFunc {
	return func(c *Config) {
		c.Retries = i
	}
}

// WithDeleteFirst will enforce deleting the resource on the server first before
// attempting to update it. This option is provided to enable replacing specific
// resources like PodDisruptionBudget. These resources can't be updated and need
// to be recreated to reconfigure.
func WithDeleteFirst() OptionFunc {
	return func(c *Config) {
		c.DeleteFirst = true
	}
}

func withName(n string) OptionFunc {
	return func(c *Config) {
		c.name = n
	}
}
