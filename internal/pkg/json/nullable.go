package json

import (
	"bytes"
	"encoding/json"
)

// Nullable represents a JSON field on a partial-update request that can be
// omitted (don't touch), explicitly null (clear the value) or set to a
// concrete value. A plain pointer can't tell "omitted" and "null" apart —
// both unmarshal to nil — which makes it impossible to clear an optional
// field via a PATCH-style request. Set is true whenever the key was present
// in the payload at all; Valid is true only when it carried a non-null value.
type Nullable[T any] struct {
	Value T
	Valid bool
	Set   bool
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	if bytes.Equal(data, []byte("null")) {
		n.Valid = false
		return nil
	}
	if err := json.Unmarshal(data, &n.Value); err != nil {
		return err
	}
	n.Valid = true
	return nil
}
