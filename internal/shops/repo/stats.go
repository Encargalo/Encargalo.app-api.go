package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/dtos"
	shopModels "Encargalo.app-api.go/internal/shops/domain/models"
	"Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type shopStatsRepo struct {
	db *bun.DB
}

func NewShopStatsRepo(db *bun.DB) ports.ShopStatsRepo {
	return &shopStatsRepo{db}
}

func (r *shopStatsRepo) GetBalance(ctx context.Context, shopID uuid.UUID) (float64, error) {
	var b shopModels.ShopBalance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", shopID, shopModels.EntityTypeShop).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		slog.Error("stats GetBalance: failed", "shopID", shopID, "error", err)
		return 0, errcustom.ErrUnexpectedError
	}
	return b.Amount, nil
}

func (r *shopStatsRepo) GetPeriodMetrics(ctx context.Context, shopID uuid.UUID, from, to time.Time) (dtos.PeriodMetrics, error) {
	var m dtos.PeriodMetrics

	var earned float64
	err := r.db.NewSelect().
		TableExpr("finance.transactions").
		ColumnExpr("COALESCE(SUM(amount), 0)").
		Where("entity_id = ? AND entity_type = ?", shopID, shopModels.EntityTypeShop).
		Where("created_at >= ? AND created_at < ?", from, to).
		Scan(ctx, &earned)
	if err != nil {
		slog.Error("stats GetPeriodMetrics: earned query failed", "shopID", shopID, "error", err)
		return m, errcustom.ErrUnexpectedError
	}
	m.TotalEarned = earned

	var rejRow struct {
		Count int     `bun:"cnt"`
		Lost  float64 `bun:"lost"`
	}
	err = r.db.NewSelect().
		TableExpr("orders.orders").
		ColumnExpr("COUNT(*) AS cnt, COALESCE(SUM((total_price - delivery_fee) * 0.9), 0) AS lost").
		Where("shop_id = ? AND status = 'Rejected'", shopID).
		Where("created_at >= ? AND created_at < ?", from, to).
		Scan(ctx, &rejRow)
	if err != nil {
		slog.Error("stats GetPeriodMetrics: rejected query failed", "shopID", shopID, "error", err)
		return m, errcustom.ErrUnexpectedError
	}
	m.OrdersRejected = rejRow.Count
	m.TotalLostRejections = rejRow.Lost

	var completed int
	err = r.db.NewSelect().
		TableExpr("orders.orders AS o").
		ColumnExpr("COUNT(DISTINCT o.id)").
		Join("JOIN finance.transactions ft ON ft.order_id = o.id AND ft.entity_type = ?", shopModels.EntityTypeShop).
		Where("o.shop_id = ? AND o.status = 'Completed'", shopID).
		Where("ft.created_at >= ? AND ft.created_at < ?", from, to).
		Scan(ctx, &completed)
	if err != nil {
		slog.Error("stats GetPeriodMetrics: completed query failed", "shopID", shopID, "error", err)
		return m, errcustom.ErrUnexpectedError
	}
	m.OrdersCompleted = completed

	return m, nil
}

func (r *shopStatsRepo) GetTopProducts(ctx context.Context, shopID uuid.UUID, from, to time.Time, limit int) ([]dtos.TopProductItem, error) {
	var rows []struct {
		ItemID       string  `bun:"item_id"`
		Name         string  `bun:"name"`
		Image        string  `bun:"image"`
		QuantitySold int     `bun:"quantity_sold"`
		TotalRevenue float64 `bun:"total_revenue"`
	}

	err := r.db.NewSelect().
		TableExpr("orders.order_items oi").
		ColumnExpr("oi.item_id::text AS item_id, oi.name, oi.image, SUM(oi.amount) AS quantity_sold, SUM(oi.total_price)::float8 AS total_revenue").
		Join("JOIN orders.orders o ON o.id = oi.order_id").
		Join("JOIN finance.transactions ft ON ft.order_id = o.id AND ft.entity_type = ?", shopModels.EntityTypeShop).
		Where("o.shop_id = ? AND o.status = 'Completed'", shopID).
		Where("ft.created_at >= ? AND ft.created_at < ?", from, to).
		GroupExpr("oi.item_id, oi.name, oi.image").
		OrderExpr("quantity_sold DESC").
		Limit(limit).
		Scan(ctx, &rows)
	if err != nil {
		slog.Error("stats GetTopProducts: query failed", "shopID", shopID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	result := make([]dtos.TopProductItem, len(rows))
	for i, row := range rows {
		result[i] = dtos.TopProductItem{
			ItemID:       row.ItemID,
			Name:         row.Name,
			Image:        row.Image,
			QuantitySold: row.QuantitySold,
			TotalRevenue: row.TotalRevenue,
		}
	}
	return result, nil
}

func (r *shopStatsRepo) GetSalesOverTime(ctx context.Context, shopID uuid.UUID, from, to time.Time, granularity string) ([]dtos.SalesDataPoint, error) {
	// granularity is controlled internally ("hour" or "day") — safe to interpolate
	// Completed orders are bucketed by the settlement transaction date (when the
	// sale was actually earned); Rejected orders have no transaction, so they
	// fall back to the order's created_at.
	dateExpr := "COALESCE(ft.created_at, o.created_at)"
	truncExpr := fmt.Sprintf("DATE_TRUNC('%s', %s)", granularity, dateExpr)

	var rows []struct {
		Date            time.Time `bun:"date"`
		Earned          float64   `bun:"earned"`
		Lost            float64   `bun:"lost"`
		OrdersCompleted int       `bun:"orders_completed"`
		OrdersRejected  int       `bun:"orders_rejected"`
	}

	err := r.db.NewSelect().
		TableExpr("orders.orders o").
		ColumnExpr(truncExpr+" AS date").
		ColumnExpr("COALESCE(SUM(ft.amount), 0) AS earned").
		ColumnExpr("COALESCE(SUM(CASE WHEN o.status = 'Rejected' THEN (o.total_price - o.delivery_fee) * 0.9 ELSE 0 END), 0) AS lost").
		ColumnExpr("COUNT(CASE WHEN o.status = 'Completed' THEN 1 END) AS orders_completed").
		ColumnExpr("COUNT(CASE WHEN o.status = 'Rejected' THEN 1 END) AS orders_rejected").
		Join("LEFT JOIN finance.transactions ft ON ft.order_id = o.id AND ft.entity_type = ?", shopModels.EntityTypeShop).
		Where("o.shop_id = ?", shopID).
		Where(dateExpr+" >= ? AND "+dateExpr+" < ?", from, to).
		GroupExpr(truncExpr).
		OrderExpr("date ASC").
		Scan(ctx, &rows)
	if err != nil {
		slog.Error("stats GetSalesOverTime: query failed", "shopID", shopID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	dateFmt := "2006-01-02"
	if granularity == "hour" {
		dateFmt = "2006-01-02 15:00"
	}

	result := make([]dtos.SalesDataPoint, len(rows))
	for i, row := range rows {
		result[i] = dtos.SalesDataPoint{
			Date:            row.Date.Format(dateFmt),
			Earned:          row.Earned,
			Lost:            row.Lost,
			OrdersCompleted: row.OrdersCompleted,
			OrdersRejected:  row.OrdersRejected,
		}
	}
	return result, nil
}

func (r *shopStatsRepo) GetCustomerMetrics(ctx context.Context, shopID uuid.UUID, from, to time.Time) (dtos.CustomerMetrics, error) {
	var row shopModels.CustomerMetricsRow

	err := r.db.NewRaw(`
		WITH customer_orders AS (
			SELECT
				o.customer_id,
				MIN(ft.created_at) OVER (PARTITION BY o.customer_id) AS first_completed_at,
				bool_or(ft.created_at >= ? AND ft.created_at < ?)
					OVER (PARTITION BY o.customer_id) AS bought_in_period
			FROM orders.orders o
			JOIN finance.transactions ft ON ft.order_id = o.id AND ft.entity_type = ?
			WHERE o.shop_id = ? AND o.status = 'Completed'
		)
		SELECT
			COUNT(DISTINCT customer_id) FILTER (WHERE bought_in_period AND first_completed_at >= ?) AS new_customers,
			COUNT(DISTINCT customer_id) FILTER (WHERE bought_in_period AND first_completed_at < ?) AS returning_customers
		FROM customer_orders
	`, from, to, shopModels.EntityTypeShop, shopID, from, from).Scan(ctx, &row)
	if err != nil {
		slog.Error("stats GetCustomerMetrics: query failed", "shopID", shopID, "error", err)
		return dtos.CustomerMetrics{}, errcustom.ErrUnexpectedError
	}

	return dtos.CustomerMetrics{
		NewCustomers:       row.NewCustomers,
		ReturningCustomers: row.ReturningCustomers,
	}, nil
}

func (r *shopStatsRepo) GetAllTransactionsInPeriod(ctx context.Context, shopID uuid.UUID, from, to time.Time) ([]dtos.ShopTransactionDTO, error) {
	var txs []shopModels.ShopTransaction
	err := r.db.NewSelect().
		Model(&txs).
		Where("entity_id = ? AND entity_type = ?", shopID, shopModels.EntityTypeShop).
		Where("created_at >= ? AND created_at < ?", from, to).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("stats GetAllTransactionsInPeriod: query failed", "shopID", shopID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	result := make([]dtos.ShopTransactionDTO, len(txs))
	for i, t := range txs {
		dto := dtos.ShopTransactionDTO{
			ID:            t.ID.String(),
			Amount:        t.Amount,
			MovementType:  t.MovementType,
			PaymentMethod: t.PaymentMethod,
			CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		}
		if t.OrderID != nil {
			s := t.OrderID.String()
			dto.OrderID = &s
		}
		result[i] = dto
	}
	return result, nil
}
