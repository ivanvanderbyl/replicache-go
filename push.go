package replicache

import "encoding/json"

type (
	PushRequest struct {
		ClientID      string     `json:"clientID"`
		Mutations     []Mutation `json:"mutations"`
		ProfileID     string     `json:"profileID"`
		PushVersion   int64      `json:"pushVersion"`
		SchemaVersion string     `json:"schemaVersion,omitempty"`
	}

	Mutation struct {
		ID   uint64          `json:"id"`
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	}
)
