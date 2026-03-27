package ipc

// PluginHostHelloRequest is sent by plugin-host subprocesses when they dial runtime IPC.
type PluginHostHelloRequest struct {
	PluginName string `json:"plugin_name"`
	Instance   int    `json:"instance"`
	Tunnel     string `json:"tunnel"`
}

func (PluginHostHelloRequest) Kind() uint16 {
	return KindPluginHostHelloRequest
}

// PluginHostHelloResponse acknowledges tunnel ownership.
type PluginHostHelloResponse struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

func (PluginHostHelloResponse) Kind() uint16 {
	return KindPluginHostHelloResponse
}
