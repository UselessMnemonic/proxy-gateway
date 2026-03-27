package api

type Activator interface {
	OnActivate(map[string]any) error
	OnDeactivate(map[string]any) error
}
