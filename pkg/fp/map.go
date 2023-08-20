package fp

import "sync"

func NewMutexMap[key comparable, value any]() MutexMap[key, value] {
	return MutexMap[key, value]{
		data: map[key]value{},
		mu:   &sync.RWMutex{},
	}
}

type MutexMap[K comparable, V any] struct {
	data map[K]V
	mu   *sync.RWMutex
}

func (m *MutexMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()
}

func (m *MutexMap[K, V]) Get(key K) (V, bool) { //nolint:ireturn
	m.mu.RLock()
	value, found := m.data[key]
	m.mu.RUnlock()

	return value, found
}
