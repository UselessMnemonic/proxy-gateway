package pluginhosts

import "proxy-gateway/internal/v2/contracts"

// TargetHintSink adapts activation hints to the target subsystem lookup interface.
type TargetHintSink struct {
	Targets contracts.TargetLookup
}

func (s TargetHintSink) HintActivate(targetName string) {
	target, ok := s.Targets.GetTarget(targetName)
	if !ok || target == nil {
		return
	}
	_ = target.Activate()
}
