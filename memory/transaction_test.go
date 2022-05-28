package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransaction(t *testing.T) {
	a := assert.New(t)

	backend := New[string]()
	backend.PutEntry("Space1", "todo-2", "Another World", 0)
	backend.PutEntry("Space1", "todo-2", "Another World", 1)

	tx := NewInMemoryTransaction[string](backend, "Space1", "2", 2)

	a.True(tx.IsEmpty())

	v := "Hello World"
	tx.Put("todo-1", &v)

	a.Equal(tx.Has("todo-1"), true)
	t1, err := tx.Get("todo-1")
	a.NoError(err)
	a.Equal("Hello World", *t1)

	// Get from backend
	a.Equal(tx.Has("todo-2"), false)
	tx.Get("todo-2")
	a.Equal(tx.Has("todo-2"), true)

	// Delete
	a.NoError(tx.Del("todo-2"))
	a.Equal(tx.Has("todo-2"), false)

	err = tx.Flush()
	a.NoError(err)

	entries := backend.GetEntries("Space1", "")
	a.Len(entries, 1)

	changes := backend.GetChangedEntries("Space1", 0)
	a.Len(changes, 2)

	if len(changes) == 0 {
		return
	}
	a.Equal("todo-1", changes[0].Key)
	a.Equal(false, changes[0].Deleted)

}
