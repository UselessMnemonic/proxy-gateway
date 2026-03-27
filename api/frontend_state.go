package api

type FrontendState int32

const (
	FrontendStateStopped FrontendState = 1
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
