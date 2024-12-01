package main

import (
	"sync"
	"time"
)

type CachedValue[T comparable] struct {
	validUntil time.Time
	value      T
}

func (v CachedValue[T]) IsValid() bool {
	return v.validUntil.Before(time.Now())
}

type CachedList[T comparable] struct {
	data []CachedValue[T]
	mut  sync.RWMutex
}

func (l *CachedList[T]) Add(value T, keep_unique bool) {
	if keep_unique && l.Contains(value) {
		return
	}

	l.mut.Lock()
	defer l.mut.Unlock()

	l.data = append(l.data, CachedValue[T]{
		validUntil: time.Now().Add(cacheValid),
		value:      value,
	})
}

func (l *CachedList[T]) Contains(value T) bool {
	l.mut.RLock()
	defer l.mut.RUnlock()

	// TODO: Consider validate against `validUntil`
	for _, val := range l.data {
		if val.value == value {
			return true
		}
	}

	return false
}

func (l *CachedList[T]) Data() []CachedValue[T] {
	l.mut.RLock()
	defer l.mut.RUnlock()

	return l.data
}

func (l *CachedList[T]) Remove(index int) {
	l.mut.Lock()
	defer l.mut.Unlock()

	if index > len(l.data) {
		return
	}

	l.data = append(l.data[:index], l.data[index+1:]...)
}

func NewCacheList[T comparable]() CachedList[T] {
	return CachedList[T]{
		data: []CachedValue[T]{},
	}
}
