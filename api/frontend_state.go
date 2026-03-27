package api

// FrontendState describes listener health and lifecycle.
type FrontendState int32

const (
	// FrontendStateStopped indicates the frontend is not accepting traffic.
	FrontendStateStopped FrontendState = 1
	// FrontendStateRunning indicates the frontend is actively listening.
	FrontendStateRunning FrontendState = 2
)

func (it FrontendState) String() string {
	switch it {
	case FrontendStateStopped:
		return "stopped"
	case FrontendStateRunning:
		return "running"
	default:
		return "invalid"
	}
}
