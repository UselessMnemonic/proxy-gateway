package api

const (
	// SymbolOnRegister is the required plugin symbol used for registration.
	SymbolOnRegister string = "OnRegister"
)

// OnRegisterFunc registers plugin-provided implementations into the runtime.
type OnRegisterFunc func(Registry) error
