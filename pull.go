package replicache

type (
	PullRequest struct {
		ClientID       string `json:"clientID"`
		Cookie         uint64 `json:"cookie"`
		LastMutationID uint64 `json:"lastMutationID"`
		ProfileID      string `json:"profileID"`
		PullVersion    int64  `json:"pullVersion"`
		SchemaVersion  string `json:"schemaVersion,omitempty"`
	}

	PullResponse[T any] struct {
		Cookie         uint64              `json:"cookie"`
		LastMutationID uint64              `json:"lastMutationID"`
		Patch          []PatchOperation[T] `json:"patch"`
	}

	PatchOperation[T any] struct {
		Op    PatchOp `json:"op"`
		Key   *string `json:"key,omitempty"`
		Value *T      `json:"value,omitempty"`
	}
)

type PatchOp string

const (
	PatchPut   PatchOp = "put"
	PatchDel   PatchOp = "del"
	PatchClear PatchOp = "clear"
)
