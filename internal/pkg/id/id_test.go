package id_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/id"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNew_IsV7(t *testing.T) {
	got := id.New()
	require.NotEqual(t, uuid.Nil, got)
	require.Equal(t, uuid.Version(7), got.Version())
}

func TestEnsure_KeepsExisting(t *testing.T) {
	existing := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	require.Equal(t, existing, id.Ensure(existing))
	require.Equal(t, uuid.Version(7), id.Ensure(uuid.Nil).Version())
}
