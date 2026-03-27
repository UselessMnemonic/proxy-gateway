package ipc

type Error struct {
	Message string
}

func (it Error) Error() string {
	return it.Message
}

func (Error) Kind() uint16 {
	return KindError
}
