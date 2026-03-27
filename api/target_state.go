package api

type TargetState int32

const (
	TargetStateInactive TargetState = 1
	TargetStateActive   TargetState = 2
	TargetStateWarming  TargetState = 3
	TargetStateDraining TargetState = 4
)

func (it TargetState) String() string {
	switch it {
	case TargetStateInactive:
		return "inactive"
	case TargetStateActive:
		return "active"
	case TargetStateWarming:
		return "warming"
	case TargetStateDraining:
		return "draining"
	default:
		return "invalid"
	}
}
