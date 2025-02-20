package cache

import (
	"sync/atomic"
)

// AtomicListRestockCache is a key value cache that stores data behind an atomic.Pointer.
// It exposes APIs to list data by key and to update data.
type AtomicListRestockCache[K comparable, S ~[]E, E any, T ~map[K]S] struct {
	data atomic.Pointer[T]
}

func newAtomicListRestockCache[K comparable, S ~[]E, T ~map[K]S, E any]() *AtomicListRestockCache[K, S, E, T] {
	return &AtomicListRestockCache[K, S, E, T]{
		data: atomic.Pointer[T]{},
	}
}

// List returns the data stored for the given key
func (c *AtomicListRestockCache[K, S, E, T]) List(key K) S {
	m := c.data.Load()
	if m == nil {
		return nil
	}
	return (*m)[key]
}

// Restock fully replaces data stored in the cache
func (c *AtomicListRestockCache[K, S, E, T]) Restock(data *T) {
	c.data.Store(data)
}
