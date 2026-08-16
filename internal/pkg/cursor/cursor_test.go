package cursor_test

import (
	"testing"
	"time"

	"Encargalo.app-api.go/internal/pkg/cursor"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	id := uuid.New()
	key := cursor.Key{CreatedAt: time.Now().UTC().Truncate(time.Microsecond), ID: id}

	encoded := cursor.Encode(key)
	decoded, err := cursor.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, key.CreatedAt, decoded.CreatedAt)
	assert.Equal(t, key.ID, decoded.ID)
}

func TestDecodeInvalid(t *testing.T) {
	_, err := cursor.Decode("not-a-cursor")
	assert.ErrorIs(t, err, cursor.ErrInvalidCursor)
}
