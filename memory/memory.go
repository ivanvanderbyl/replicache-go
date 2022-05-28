package memory

import (
	"fmt"
	"sort"
	"time"

	"github.com/zyedidia/generic"
	"github.com/zyedidia/generic/btree"
)

var ErrNotFound = fmt.Errorf("not found")

type (
	MemoryBackend[T any] struct {
		entries *btree.Tree[string, *Entry[T]]
		spaces  *btree.Tree[string, *Space]
		clients *btree.Tree[string, *Client]
	}

	Entry[T any] struct {
		SpaceID        string
		Key            string
		Value          T
		Deleted        bool
		Version        uint64
		LastModifiedAt time.Time
	}

	Client struct {
		ID             string
		LastMutationID uint64
		LastModifiedAt time.Time
	}

	Space struct {
		ID             string
		Version        uint64
		LastModifiedAt time.Time
	}
)

func New[T any]() *MemoryBackend[T] {
	return &MemoryBackend[T]{
		entries: btree.New[string, *Entry[T]](generic.Less[string]),
		spaces:  btree.New[string, *Space](generic.Less[string]),
		clients: btree.New[string, *Client](generic.Less[string]),
	}
}

func (t *MemoryBackend[T]) PutEntry(spaceID string, key string, value T, version uint64) error {
	id := makeKey(spaceID, key)

	if entry, ok := t.entries.Get(id); ok {
		entry.LastModifiedAt = time.Now()
		entry.Version = version
		entry.Value = value
		entry.Deleted = false
		t.entries.Put(id, entry)
		return nil
	}

	t.entries.Put(id, &Entry[T]{
		SpaceID:        spaceID,
		Key:            key,
		Value:          value,
		Deleted:        false,
		Version:        version,
		LastModifiedAt: time.Now(),
	})
	return nil
}

func (t *MemoryBackend[T]) GetEntry(spaceID string, key string) (*T, error) {
	entry, ok := t.entries.Get(makeKey(spaceID, key))
	if !ok {
		return nil, ErrNotFound
	}

	if entry.Deleted {
		return nil, ErrNotFound
	}

	return &entry.Value, nil
}

func (t *MemoryBackend[T]) DelEntry(spaceID string, key string, version uint64) error {
	entry, ok := t.entries.Get(makeKey(spaceID, key))
	if !ok {
		return ErrNotFound
	}

	entry.Deleted = true
	entry.LastModifiedAt = time.Now()
	t.entries.Put(makeKey(spaceID, key), entry)
	return nil
}

func (t *MemoryBackend[T]) GetEntries(spaceID string, fromKey string) []*Entry[T] {
	entries := make([]*Entry[T], 0)

	t.entries.Each(func(key string, val *Entry[T]) {
		if val.Key >= fromKey && spaceID == val.SpaceID && !val.Deleted {
			entries = append(entries, val)
		}
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}

func (t *MemoryBackend[T]) GetCookie(spaceID string) (uint64, bool) {
	space, ok := t.spaces.Get(spaceID)
	if !ok {
		return 0, false
	}
	return space.Version, true
}

func (t *MemoryBackend[T]) SetCookie(spaceID string, version uint64) {
	if space, ok := t.spaces.Get(spaceID); ok {
		space.LastModifiedAt = time.Now()
		space.Version = version
		return
	}

	t.spaces.Put(spaceID, &Space{
		ID:             spaceID,
		Version:        version,
		LastModifiedAt: time.Now(),
	})
}

func (t *MemoryBackend[T]) GetLastMutationID(clientID string) (uint64, bool) {
	client, ok := t.clients.Get(clientID)
	if !ok {
		return 0, false
	}
	return client.LastMutationID, true
}

func (t *MemoryBackend[T]) SetLastMutationID(clientID string, lastMutationID uint64) {
	client, ok := t.clients.Get(clientID)
	if !ok {
		t.clients.Put(clientID, &Client{
			ID:             clientID,
			LastMutationID: lastMutationID,
			LastModifiedAt: time.Now(),
		})
		return
	}
	client.LastMutationID = lastMutationID
	client.LastModifiedAt = time.Now()
}

func (t *MemoryBackend[T]) GetChangedEntries(spaceID string, prevVersion uint64) []*Entry[T] {
	entries := make([]*Entry[T], 0)
	t.entries.Each(func(key string, val *Entry[T]) {
		if val.SpaceID == spaceID && val.Version > prevVersion {
			entries = append(entries, val)
		}
	})
	return entries
}

func (t *MemoryBackend[T]) Size() int {
	return t.entries.Size()
}

func makeKey(spaceID string, key string) string {
	return spaceID + ":" + key
}
