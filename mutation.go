package replicache

type (
	Backend[T any] interface {
		GetEntry(spaceID string, key string) (*T, error)
		PutEntry(spaceID string, key string, entry T, version uint64) error
		DelEntry(spaceID string, key string, version uint64) error
	}
)
