package cursor

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("cursor inválido")

// Key identifica la posición en una lista ordenada por created_at DESC, id DESC.
type Key struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func Encode(k Key) string {
	b, _ := json.Marshal(k)
	return base64.RawURLEncoding.EncodeToString(b)
}

func Decode(raw string) (Key, error) {
	if raw == "" {
		return Key{}, ErrInvalidCursor
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Key{}, ErrInvalidCursor
	}
	var k Key
	if err := json.Unmarshal(b, &k); err != nil {
		return Key{}, ErrInvalidCursor
	}
	if k.ID == uuid.Nil || k.CreatedAt.IsZero() {
		return Key{}, ErrInvalidCursor
	}
	return k, nil
}
