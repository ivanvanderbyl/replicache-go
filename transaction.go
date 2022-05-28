package replicache

type (
	ReadWriteTransaction[T any] interface {
		ReadTransaction[T]
		WriteTransaction[T]
	}

	WriteTransaction[T any] interface {
		Put(key string, value *T) error
		Del(key string) error
		Flush() error
	}

	ReadTransaction[T any] interface {
		Get(key string) (*T, error)
		Has(key string) bool
		IsEmpty() bool
	}

	Value[T any] struct {
		Value *T
		Dirty bool
	}

	Entry struct {
		Key     string
		Value   any
		Deleted bool
		SpaceID string
		Version uint64
	}
)
