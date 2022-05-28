package memory

import (
	"sync"

	"github.com/airheartdev/replicache"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/zyedidia/generic"
	"github.com/zyedidia/generic/btree"
)

type InMemoryTransaction[T any] struct {
	cache    *btree.Tree[string, replicache.Value[T]]
	spaceID  string
	clientID string
	version  uint64
	backend  replicache.Backend[T]
	// Executor func(WriteTransaction) error
	mu *sync.Mutex
}

func NewInMemoryTransaction[T any](backend replicache.Backend[T], spaceID string, clientID string, version uint64) replicache.ReadWriteTransaction[T] {
	return &InMemoryTransaction[T]{
		backend:  backend,
		mu:       &sync.Mutex{},
		spaceID:  spaceID,
		clientID: clientID,
		version:  version,
		cache:    btree.New[string, replicache.Value[T]](generic.Less[string]),
	}
}

var _ replicache.WriteTransaction[any] = &InMemoryTransaction[any]{}

func (t *InMemoryTransaction[T]) Put(key string, value *T) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache.Put(key, replicache.Value[T]{Value: value, Dirty: true})
	return nil
}

func (t *InMemoryTransaction[T]) Del(key string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.cache.Get(key)
	if ok {
		t.cache.Put(key, replicache.Value[T]{Dirty: true, Value: nil})
	}
	return nil
}

func (t *InMemoryTransaction[T]) Get(key string) (*T, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	val, ok := t.cache.Get(key)
	if ok {
		return val.Value, nil
	}

	entry, err := t.backend.GetEntry(t.spaceID, key)
	if err != nil {
		return nil, err
	}

	t.cache.Put(key, replicache.Value[T]{Value: entry, Dirty: false})
	return entry, nil
}

func (t *InMemoryTransaction[T]) Has(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	val, ok := t.cache.Get(key)
	return ok && val.Value != nil
}

func (t *InMemoryTransaction[T]) IsEmpty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cache.Size() == 0
}

func (t *InMemoryTransaction[T]) Flush() error {
	backend := t.backend
	t.mu.Lock()
	defer t.mu.Unlock()

	var errs error
	t.cache.Each(func(key string, val replicache.Value[T]) {
		if !val.Dirty {
			return
		}

		if val.Value == nil {
			err := backend.DelEntry(t.spaceID, key, t.version)
			if err != nil {
				multierror.Append(errs, err)
			}
		} else {
			err := backend.PutEntry(t.spaceID, key, *val.Value, t.version)
			if err != nil {
				multierror.Append(errs, err)
			}
		}
	})

	return errs
}
