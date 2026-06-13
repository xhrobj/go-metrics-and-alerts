package pool

import (
	"errors"
	"sync"
)

var errNilNewObject = errors.New("new object function is nil")

// Resetter описывает объект, состояние которого можно сбросить.
type Resetter interface {
	Reset()
}

// Pool хранит переиспользуемые объекты одного типа.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт Pool для объектов одного типа.
// Возвращает ошибку, если функция создания объектов равна nil.
func New[T Resetter](newObject func() T) (*Pool[T], error) {
	if newObject == nil {
		return nil, errNilNewObject
	}

	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newObject()
			},
		},
	}, nil
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put сбрасывает состояние объекта и возвращает его в пул.
func (p *Pool[T]) Put(object T) {
	object.Reset()
	p.pool.Put(object)
}
