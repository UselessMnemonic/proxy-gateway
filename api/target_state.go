package api

// TargetState describes target activation lifecycle.
type TargetState int32

const (
	// TargetStateInactive means the target is not accepting forwarded traffic.
	TargetStateInactive TargetState = 1
	// TargetStateActive means the target is actively serving forwarded traffic.
	TargetStateActive TargetState = 2
	// TargetStateWarming means activation is in progress.
	TargetStateWarming TargetState = 3
	// TargetStateDraining means deactivation is in progress.
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
