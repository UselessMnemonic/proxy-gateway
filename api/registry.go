package api

// Registry exposes plugin registration hooks for activators and interceptors.
type Registry interface {
	// RegisterActivator registers an activator implementation by kind.
	RegisterActivator(string, Activator) error
	// RegisterInterceptor registers an interceptor implementation by kind.
	RegisterInterceptor(string, Interceptor) error
}
