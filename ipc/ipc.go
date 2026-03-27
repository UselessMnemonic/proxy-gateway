package ipc

import (
	"encoding/gob"
	"fmt"
)

type Value interface {
	Kind() uint16
}

type Wrapper struct {
	Kind uint16
	Body Value
}

func WrapValue(value Value) Wrapper {
	return Wrapper{
		Kind: value.Kind(),
		Body: value,
	}
}

func UnwrapValue[T Value](it Wrapper) (T, error) {
	var t T
	if it.Kind != t.Kind() {
		return t, fmt.Errorf("expected kind %d is but got %d", t.Kind(), it.Kind)
	}

	t, ok := it.Body.(T)
	if !ok {
		return t, fmt.Errorf("expected type %T but got %T", t, it.Body)
	}

	return t, nil
}

const (
	KindNone = iota
	KindError
	KindStatusRequest
	KindStatusResponse
)

func RegisterGobTypes() {
	gob.Register(Error{})
	gob.Register(StatusRequest{})
	gob.Register(StatusResponse{})
}
