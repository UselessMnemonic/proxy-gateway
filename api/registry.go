package api

type Registry interface {
	RegisterActivator(string, Activator) error
	RegisterInterceptor(string, Interceptor) error
}
