package e2e_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	ridersmodels "Encargalo.app-api.go/internal/riders/domain/models"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestSchema_PaymentsReviewsReplaceOrdersTable(t *testing.T) {
	require.True(t, relationExists(t, "payments.reviews"))
	require.False(t, relationExists(t, "orders.payment_reviews"))

	reviewCols := columns(t, "payments", "reviews")
	require.Contains(t, reviewCols, "reference_id")
	require.Contains(t, reviewCols, "reference_type")
	require.NotContains(t, reviewCols, "order_id")

	txCols := columns(t, "finance", "transactions")
	require.Contains(t, txCols, "reference_id")
	require.Contains(t, txCols, "reference_type")
	require.Contains(t, txCols, "order_id")

	refundCols := columns(t, "finance", "balance_refund_requests")
	require.Contains(t, refundCols, "reference_id")
	require.Contains(t, refundCols, "reference_type")
	require.NotContains(t, refundCols, "order_id")
}

func TestSchema_RiderTransactionStillWritesOrderID(t *testing.T) {
	orderID := uuid.New()
	row := &ridersmodels.Transaction{
		ID:           uuid.New(),
		EntityID:     uuid.New(),
		EntityType:   "rider",
		OrderID:      &orderID,
		Amount:       12.5,
		Currency:     "VES",
		MovementType: "ride_bank",
	}
	_, err := db.NewInsert().Model(row).Exec(context.Background())
	require.NoError(t, err)

	var got ledgerRow
	err = db.NewSelect().Model(&got).Where("id = ?", row.ID).Scan(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got.OrderID)
	require.Equal(t, orderID, *got.OrderID)
	require.Nil(t, got.ReferenceID)
	require.Nil(t, got.ReferenceType)
}

func TestSchema_DownRestoresOrderScopedColumns(t *testing.T) {
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, `CREATE DATABASE encargalo_down`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = sqlDB.ExecContext(context.Background(), `DROP DATABASE IF EXISTS encargalo_down WITH (FORCE)`)
	})

	downDB := openSQL(swapDatabase(baseDSN, "encargalo_down"))
	t.Cleanup(func() { _ = downDB.Close() })

	require.NoError(t, migrateUp(downDB))
	require.NoError(t, goose.DownTo(downDB, "migrations", 73))

	require.False(t, relationExistsOn(t, downDB, "payments.reviews"))
	require.True(t, relationExistsOn(t, downDB, "orders.payment_reviews"))

	txCols := columnsOn(t, downDB, "finance", "transactions")
	require.NotContains(t, txCols, "reference_id")
	require.NotContains(t, txCols, "reference_type")
	require.Contains(t, txCols, "order_id")

	refundCols := columnsOn(t, downDB, "finance", "balance_refund_requests")
	require.Contains(t, refundCols, "order_id")
	require.NotContains(t, refundCols, "reference_id")
	require.NotContains(t, refundCols, "reference_type")
}

func TestRoutes_StaffReviewsMovedToPayments(t *testing.T) {
	root, err := findModuleRoot()
	require.NoError(t, err)
	ordersSrc, err := os.ReadFile(filepath.Join(root, "api/router/groups/orders.go"))
	require.NoError(t, err)
	require.NotContains(t, string(ordersSrc), "staff/reviews")

	resp, err := httpServer.Client().Get(httpServer.URL + "/payments/staff/reviews")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	// El GET plano no es un upgrade: Echo no responde 404, la ruta existe.
	require.NotEqual(t, 404, resp.StatusCode)

	missing, err := httpServer.Client().Get(httpServer.URL + "/orders/staff/reviews")
	require.NoError(t, err)
	defer func() { _ = missing.Body.Close() }()
	require.Equal(t, 404, missing.StatusCode)
}

func relationExists(t *testing.T, name string) bool {
	t.Helper()
	return relationExistsOn(t, sqlDB, name)
}

func relationExistsOn(t *testing.T, instance *sql.DB, name string) bool {
	t.Helper()
	var rel sql.NullString
	err := instance.QueryRowContext(context.Background(), `SELECT to_regclass($1)`, name).Scan(&rel)
	require.NoError(t, err)
	return rel.Valid
}

func columns(t *testing.T, schema, table string) []string {
	t.Helper()
	return columnsOn(t, sqlDB, schema, table)
}

func columnsOn(t *testing.T, instance *sql.DB, schema, table string) []string {
	t.Helper()
	rows, err := instance.QueryContext(context.Background(), `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
	`, schema, table)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		out = append(out, name)
	}
	require.NoError(t, rows.Err())
	return out
}
