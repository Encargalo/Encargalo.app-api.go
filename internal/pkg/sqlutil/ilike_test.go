package sqlutil_test

import (
	"testing"

	"Encargalo.app-api.go/internal/pkg/sqlutil"
	"github.com/stretchr/testify/assert"
)

func TestILIKEContains_EscapesWildcards(t *testing.T) {
	assert.Equal(t, `%100\%\_%`, sqlutil.ILIKEContains(`100%_`))
}
