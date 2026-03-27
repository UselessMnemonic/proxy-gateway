package api

const (
	SymbolOnRegister string = "OnRegister"
)

type OnRegisterFunc func(Registry) error
