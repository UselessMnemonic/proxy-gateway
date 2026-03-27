package ipc

import "net/netip"

// ExternalFrontendRegisterRequest lets an external process announce a frontend listener.
type ExternalFrontendRegisterRequest struct {
	Name       string         `json:"name"`
	Protocol   string         `json:"protocol"`
	Listen     netip.AddrPort `json:"listen"`
	Target     string         `json:"target"`
	TargetPort string         `json:"target_port"`
}

func (ExternalFrontendRegisterRequest) Kind() uint16 {
	return KindExternalFrontendRegisterRequest
}

// ExternalFrontendRegisterResponse communicates accepted registration state.
type ExternalFrontendRegisterResponse struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

func (ExternalFrontendRegisterResponse) Kind() uint16 {
	return KindExternalFrontendRegisterResponse
}

// ExternalFrontendHeartbeatRequest keeps an external frontend attached.
type ExternalFrontendHeartbeatRequest struct {
	Name string `json:"name"`
}

func (ExternalFrontendHeartbeatRequest) Kind() uint16 {
	return KindExternalFrontendHeartbeatRequest
}

// ExternalFrontendHeartbeatResponse indicates whether lease remains valid.
type ExternalFrontendHeartbeatResponse struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

func (ExternalFrontendHeartbeatResponse) Kind() uint16 {
	return KindExternalFrontendHeartbeatResponse
}
