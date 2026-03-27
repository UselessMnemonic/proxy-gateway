package signals

import (
	"context"
	"sync"
)

type Subscription[T any] struct {
	obs    *Observable[T]
	ctx    context.Context
	cancel context.CancelFunc
	values chan T
	once   sync.Once
}

func (s *Subscription[T]) Values() <-chan T {
	return s.values
}

func (s *Subscription[T]) Cancel() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		s.cancel()
		if s.obs != nil {
			s.obs.remove(s)
		}
	})
}

type Observable[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu   sync.RWMutex
	subs map[*Subscription[T]]struct{}
}

func NewObservable[T any](parent context.Context) *Observable[T] {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	o := &Observable[T]{
		ctx:    ctx,
		cancel: cancel,
		subs:   make(map[*Subscription[T]]struct{}),
	}
	go func() {
		<-o.ctx.Done()
		o.closeAll()
	}()
	return o
}

func (o *Observable[T]) Subscribe(buffer int) *Subscription[T] {
	if buffer < 0 {
		buffer = 0
	}
	subCtx, cancel := context.WithCancel(o.ctx)
	sub := &Subscription[T]{
		obs:    o,
		ctx:    subCtx,
		cancel: cancel,
		values: make(chan T, buffer),
	}

	o.mu.Lock()
	if o.ctx.Err() != nil {
		o.mu.Unlock()
		sub.Cancel()
		return sub
	}
	o.subs[sub] = struct{}{}
	o.mu.Unlock()

	go func() {
		<-subCtx.Done()
		sub.Cancel()
	}()

	return sub
}

func (o *Observable[T]) Emit(value T) {
	o.mu.RLock()
	subs := make([]*Subscription[T], 0, len(o.subs))
	for sub := range o.subs {
		subs = append(subs, sub)
	}
	o.mu.RUnlock()

	for _, sub := range subs {
		select {
		case <-sub.ctx.Done():
			continue
		case sub.values <- value:
		}
	}
}

func (o *Observable[T]) Cancel() {
	if o == nil {
		return
	}
	o.cancel()
}

func (o *Observable[T]) remove(sub *Subscription[T]) {
	if o == nil || sub == nil {
		return
	}
	o.mu.Lock()
	_, exists := o.subs[sub]
	if exists {
		delete(o.subs, sub)
	}
	o.mu.Unlock()
	if exists {
		close(sub.values)
	}
}

func (o *Observable[T]) closeAll() {
	o.mu.Lock()
	subs := make([]*Subscription[T], 0, len(o.subs))
	for sub := range o.subs {
		subs = append(subs, sub)
	}
	o.subs = make(map[*Subscription[T]]struct{})
	o.mu.Unlock()

	for _, sub := range subs {
		sub.once.Do(func() {
			sub.cancel()
			close(sub.values)
		})
	}
}
