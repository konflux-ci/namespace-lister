package cache

import (
	"slices"
	"sync/atomic"
)

// AtomicListRestockCache is a key value cache that stores data behind an atomic.Pointer.
// It exposes APIs to list data by key and to update data.
type AtomicListRestockCache[K comparable, S ~[]E, E any, T ~map[K]S, I comparable] struct {
	data   atomic.Pointer[T]
	idFunc func(E) I
}

func newAtomicListRestockCache[K comparable, S ~[]E, T ~map[K]S, I comparable, E any](idFunc func(E) I) *AtomicListRestockCache[K, S, E, T, I] {
	return &AtomicListRestockCache[K, S, E, T, I]{
		data:   atomic.Pointer[T]{},
		idFunc: idFunc,
	}
}

// Restock fully replaces data stored in the cache
func (c *AtomicListRestockCache[K, S, E, T, I]) Restock(data *T) {
	c.data.Store(data)
}

func (c *AtomicListRestockCache[K, S, E, T, I]) List(keys ...K) S {
	switch len(keys) {
	case 0:
		return nil
	case 1:
		return c.list(keys[0])
	default:
		return c.listAll(keys...)
	}
}

// List returns the data stored for the given key
func (c *AtomicListRestockCache[K, S, E, T, I]) list(key K) S {
	m := c.data.Load()
	if m == nil {
		return nil
	}
	return (*m)[key]
}

func (c *AtomicListRestockCache[K, S, E, T, I]) listAll(keys ...K) S {
	// retrieve cached data
	m := c.data.Load()

	// worst case scenario: namespaces from provided users are different among them
	maxSize := 0

	// store namespaces for each subject in a map
	// so we'll avoid duplicates
	d := map[I]E{}
	for _, k := range keys {
		nn, ok := (*m)[k]
		if !ok {
			continue
		}

		maxSize += len(nn)
		for _, n := range nn {
			// calculate id for resource
			k := c.idFunc(n)
			// if not already present, add the resource
			if _, ok := d[k]; !ok {
				d[k] = n
			}
		}
	}

	// collapse the map to a slice
	r := make([]E, 0, maxSize)
	for _, n := range d {
		r = append(r, n)
	}

	// remove exceeding capacity and return the list
	return slices.Clip(r)
}
