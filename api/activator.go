package api

// Activator controls lifecycle transitions for a target backend.
type Activator interface {
	// OnActivate is called when a target transitions into active service.
	OnActivate(map[string]any) error
	// OnDeactivate is called when a target transitions out of active service.
	OnDeactivate(map[string]any) error
}
