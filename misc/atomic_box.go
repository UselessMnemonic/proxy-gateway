package misc

import (
	"sync/atomic"
)

type box[T any] struct {
	value T
}

// AtomicBox wraps atomic.Value and adds explicit support for an empty value.
// It is safe for concurrent use.
type AtomicBox[T any] struct {
	value atomic.Value
}

func (b *AtomicBox[T]) Store(next T) {
	b.value.Store(box[T]{value: next})
}

func (b *AtomicBox[T]) Load() T {
	var wrapped box[T]

	raw := b.value.Load()
	if raw == nil {
		return wrapped.value
	}

	wrapped = raw.(box[T])
	return wrapped.value
}
