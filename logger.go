package kubekit

import "log"

// Logging represents the interface kubekit uses for its logger instance.
type Logging interface {
	Infof(string, ...interface{})
}

// Logger is the implementation of Logging which kubekit uses to log all it's
// information.
var Logger Logging = &defaultLogger{}

type defaultLogger struct{}

func (*defaultLogger) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}
