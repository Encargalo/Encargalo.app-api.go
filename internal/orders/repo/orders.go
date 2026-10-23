package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/schema"

	items "Encargalo.app-api.go/internal/products/domain/models"
)

const pgUniqueViolation = "23505"

// firstOrderDeliveryConstraint impide más de un pedido vivo o completado con envío gratis por cliente.
const firstOrderDeliveryConstraint = "uq_orders_first_order_delivery"

type orders struct {
	db *bun.DB
}

func NewOrdersRepo(db *bun.DB) ports.OrdersRepo {
	return &orders{db}
}

func (o *orders) CreateOrder(ctx context.Context, order *models.Order) (*models.Order, error) {
	err := o.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {

		lastNum, err := getLastOrderNumber(ctx, tx, order.BatchID)
		if err != nil {
			slog.Error("CreateOrder: failed to get last order number", "batchID", order.BatchID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		order.Number = lastNum + 1

		if _, err := tx.NewInsert().Model(order).Exec(ctx); err != nil {
			var pgErr pgdriver.Error
			if errors.As(err, &pgErr) && pgErr.Field('C') == pgUniqueViolation && pgErr.Field('n') == firstOrderDeliveryConstraint {
				return errcustom.ErrFirstOrderDeliveryUsed
			}
			slog.Error("CreateOrder: failed to insert order", "orderID", order.ID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if _, err := tx.NewInsert().Model(&order.ItemsOrders).Exec(ctx); err != nil {
			slog.Error("CreateOrder: failed to insert items_orders", "orderID", order.ID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		for _, v := range order.ItemsOrders {
			if len(v.Additions) == 0 {
				continue
			}

			if _, err := tx.NewInsert().Model(&v.Additions).Exec(ctx); err != nil {
				slog.Error("CreateOrder: failed to insert items_orders_additions", "orderID", order.ID, "error", err)
				return errcustom.ErrUnexpectedError
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (o *orders) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {

	order := new(models.Order)

	if err := o.db.NewSelect().Model(order).Order("created_at ASC").
		Relation("ItemsOrders").
		Relation("ItemsOrders.Additions").
		Where("id = ?", orderID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("SearchOrdersByID: failed to find order", "orderID", orderID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return order, nil

}

func (o *orders) SearchShopByID(ctx context.Context, shopID uuid.UUID) (*models.ShopOrder, error) {
	shop := new(models.ShopOrder)
	if err := o.db.NewSelect().Model(shop).Where("id = ?", shopID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("SearchShopByID: failed to find shop", "shopID", shopID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return shop, nil
}

func (o *orders) SearchItemsByID(ctx context.Context, ItemsID []uuid.UUID) (items.Items, error) {

	var items items.Items

	err := o.db.NewSelect().Model(&items).
		// Calificado con "item." porque el LEFT JOIN a item_discounts (alias
		// "discount") también tiene columna "id"; sin calificar, Postgres
		// reporta "column reference \"id\" is ambiguous" (SQLSTATE=42702).
		Where("item.id in (?)", bun.In(ItemsID)).
		// La condición de descuento activo debe ir en el ON del LEFT JOIN
		// (AdditionalJoinOnConditions), no en un .Where() dentro del callback
		// de Relation(): eso agrega la condición al WHERE de la consulta y,
		// como NULL = true es falso, convierte el LEFT JOIN en INNER JOIN de
		// facto, excluyendo del resultado a todo item SIN descuento activo
		// (el alias del JOIN es el nombre del campo Go de la relación
		// Discount -> discount, no el nombre de la tabla).
		RelationWithOpts("Discount", bun.RelationOpts{
			AdditionalJoinOnConditions: []schema.QueryWithArgs{
				bun.SafeQuery("discount.is_active = ? AND discount.deleted_at IS NULL", true),
			},
		}).
		Relation("Category").
		Scan(ctx, &items)
	if err != nil {
		fmt.Println(err)
	}

	uniqueIDs := make(map[uuid.UUID]struct{})
	for _, id := range ItemsID {
		uniqueIDs[id] = struct{}{}
	}

	if len(items) != len(uniqueIDs) {
		return nil, errcustom.ErrItemsNotFound
	}

	return items, nil
}

func (o *orders) SearchAdditionsByID(ctx context.Context, additionID []uuid.UUID) ([]models.Addition, error) {
	if len(additionID) == 0 {
		return nil, nil
	}

	var rows []models.Addition
	if err := o.db.NewSelect().
		TableExpr("products.items i").
		ColumnExpr("i.id").
		ColumnExpr("i.shop_id").
		ColumnExpr("i.name").
		ColumnExpr("i.price").
		Where("i.id IN (?)", bun.In(additionID)).
		Where("i.deleted_at IS NULL").
		Scan(ctx, &rows); err != nil {
		slog.Error("SearchAdditionsByID: failed to load additions from items", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	uniqueIDs := make(map[uuid.UUID]struct{})
	for _, id := range additionID {
		uniqueIDs[id] = struct{}{}
	}

	if len(rows) != len(uniqueIDs) {
		return nil, errcustom.ErrItemsNotFound
	}

	return rows, nil
}

func (o *orders) SearchFlavorsForItem(ctx context.Context, itemID uuid.UUID) ([]models.Flavor, error) {
	var row struct {
		Details struct {
			Flavors []models.Flavor `json:"flavors"`
		} `bun:"details,type:jsonb"`
	}

	if err := o.db.NewSelect().
		TableExpr("products.items").
		Column("details").
		Where("id = ?", itemID).
		Scan(ctx, &row); err != nil {
		slog.Error("SearchFlavorsForItem: failed to load item flavors", "itemID", itemID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	if len(row.Details.Flavors) == 0 {
		return nil, errcustom.ErrItemsNotFound
	}

	return row.Details.Flavors, nil
}

func (o *orders) SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID, isShop bool) ([]models.Order, error) {
	var orders []models.Order

	var q *bun.SelectQuery

	if isShop {
		openBatch := o.db.NewSelect().
			TableExpr("business.batches").
			ColumnExpr("id").
			Where("shop_id = ?", criteria).
			Where("opened = ?", true).
			Limit(1)

		q = o.db.NewSelect().
			Model(&orders).
			Where("shop_id = ?", criteria).
			Where("batch_id = (?)", openBatch).
			Where("status NOT IN (?)", bun.In([]string{
				"Payment Review",
			}))
	} else {
		q = o.db.NewSelect().
			Model(&orders).
			Where("customer_id = ?", criteria).
			Where("status IN (?)", bun.In([]string{
				"Payment Review",
				"Pending",
				"In Preparation",
				"Ready",
				"On The Way",
			}))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, errcustom.ErrUnexpectedError
	}

	if len(orders) == 0 {
		return nil, errcustom.ErrNotFound
	}

	return orders, nil
}

func (o *orders) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	q := o.db.NewUpdate().Model(&models.Order{}).
		Set("status = ?", status).
		Where("id = ?", orderID)

	if status == "Cancelled" {
		q = q.Set("canceled_at = ?", time.Now())
	}

	if _, err := q.Exec(ctx); err != nil {
		slog.Error("UpdateOrderStatus: failed to update order status",
			"orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (o *orders) SavePickupCode(ctx context.Context, orderID uuid.UUID, code string) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("pickup_code = ?", code).
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("SavePickupCode: failed to save pickup code", "orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (o *orders) SaveDeliveryCode(ctx context.Context, orderID uuid.UUID, code string) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("delivery_code = ?", code).
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("SaveDeliveryCode: failed to save delivery code", "orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (o *orders) MarkPickupCodeUsed(ctx context.Context, orderID uuid.UUID) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("pickup_code_used_at = NOW()").
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("MarkPickupCodeUsed: failed to mark pickup code used", "orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (o *orders) MarkDeliveryCodeUsed(ctx context.Context, orderID uuid.UUID) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("delivery_code_used_at = NOW()").
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("MarkDeliveryCodeUsed: failed to mark delivery code used", "orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (o *orders) AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error {
	res, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("rider_id = ?", riderID).
		Set("rider_accepted_at = NOW()").
		Where("id = ? AND rider_id IS NULL", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("AssignRider: failed to assign rider", "orderID", orderID, "riderID", riderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("AssignRider: failed to read rows affected", "orderID", orderID, "riderID", riderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	if rows == 0 {
		return errcustom.ErrOrderAlreadyAssigned
	}
	return nil
}

func (o *orders) SearchAvailableOrders(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order
	err := o.db.NewSelect().
		Model(&orders).
		Where("rider_id IS NULL").
		Where("status IN (?)", bun.In([]string{"In Preparation", "Ready"})).
		Scan(ctx)
	if err != nil {
		return nil, errcustom.ErrUnexpectedError
	}
	if len(orders) == 0 {
		return nil, errcustom.ErrNotFound
	}
	return orders, nil
}

func (o *orders) SearchActiveOrdersByRiderID(ctx context.Context, riderID uuid.UUID) ([]models.Order, error) {
	var orders []models.Order
	err := o.db.NewSelect().
		Model(&orders).
		Where("rider_id = ?", riderID).
		Where("status IN (?)", bun.In([]string{"In Preparation", "Ready", "On The Way"})).
		Scan(ctx)
	if err != nil {
		return nil, errcustom.ErrUnexpectedError
	}
	if len(orders) == 0 {
		return nil, errcustom.ErrNotFound
	}
	return orders, nil
}

func (o *orders) HasBillableOrder(ctx context.Context, customerID uuid.UUID) (bool, error) {
	exists, err := o.db.NewSelect().
		Model((*models.Order)(nil)).
		Where("customer_id = ?", customerID).
		Where("status NOT IN (?)", bun.In([]string{string(models.StatusCancelled), string(models.StatusRejected)})).
		Exists(ctx)
	if err != nil {
		slog.Error("orders HasBillableOrder failed", "customerID", customerID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return exists, nil
}

func (o *orders) SearchOrderHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Order, error) {
	var result []models.Order
	err := o.db.NewSelect().
		Model(&result).
		Where("customer_id = ?", customerID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, errcustom.ErrUnexpectedError
	}
	return result, nil
}

func (o *orders) ListStatusHistory(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]models.StatusChange, error) {
	result := make(map[uuid.UUID][]models.StatusChange, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}
	var rows []models.StatusChange
	err := o.db.NewSelect().
		Model(&rows).
		Where("order_id IN (?)", bun.In(orderIDs)).
		OrderExpr("at ASC, id ASC").
		Scan(ctx)
	if err != nil {
		slog.Error("ListStatusHistory failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	for _, r := range rows {
		result[r.OrderID] = append(result[r.OrderID], r)
	}
	return result, nil
}

func (o *orders) UpdateOrderDetails(ctx context.Context, orderID uuid.UUID, details []byte) error {
	// []byte se enlaza como bytea; jsonb rechaza ese formato. json.RawMessage
	// se escribe como texto JSON.
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("details = ?", json.RawMessage(details)).
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("UpdateOrderDetails failed", "orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (o *orders) MarkPendingAt(ctx context.Context, orderID uuid.UUID, at time.Time) error {
	_, err := o.db.NewUpdate().Model((*models.Order)(nil)).
		Set("pending_at = ?", at).
		Set("pending_alert_minute = ?", 0).
		Where("id = ?", orderID).
		Exec(ctx)
	if err != nil {
		slog.Error("MarkPendingAt failed", "orderID", orderID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (o *orders) ListStalePending(ctx context.Context, olderThan time.Time) ([]models.Order, error) {
	var orders []models.Order
	err := o.db.NewSelect().
		Model(&orders).
		Where("status = ?", string(models.StatusPending)).
		Where("pending_at IS NOT NULL").
		Where("pending_at <= ?", olderThan).
		OrderExpr("pending_at ASC").
		Scan(ctx)
	if err != nil {
		slog.Error("ListStalePending failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return orders, nil
}

func (o *orders) RejectIfPending(ctx context.Context, orderID uuid.UUID) (bool, error) {
	res, err := o.db.NewUpdate().Model((*models.Order)(nil)).
		Set("status = ?", string(models.StatusRejected)).
		Where("id = ?", orderID).
		Where("status = ?", string(models.StatusPending)).
		Exec(ctx)
	if err != nil {
		slog.Error("RejectIfPending failed", "orderID", orderID, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (o *orders) ListPendingUnassignedForAlert(ctx context.Context, olderThan time.Time) ([]models.Order, error) {
	var orders []models.Order
	err := o.db.NewSelect().
		Model(&orders).
		Where("status = ?", string(models.StatusPending)).
		Where("pending_at IS NOT NULL").
		Where("pending_at <= ?", olderThan).
		Where("pending_alert_minute < ?", models.PendingUnassignedMaxAlertMinute).
		OrderExpr("pending_at ASC").
		Scan(ctx)
	if err != nil {
		slog.Error("ListPendingUnassignedForAlert failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return orders, nil
}

func (o *orders) MarkPendingAlertMinute(ctx context.Context, orderID uuid.UUID, minute int) (bool, error) {
	res, err := o.db.NewUpdate().Model((*models.Order)(nil)).
		Set("pending_alert_minute = ?", minute).
		Where("id = ?", orderID).
		Where("status = ?", string(models.StatusPending)).
		Where("pending_alert_minute < ?", minute).
		Exec(ctx)
	if err != nil {
		slog.Error("MarkPendingAlertMinute failed", "orderID", orderID, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func getLastOrderNumber(ctx context.Context, tx bun.Tx, batchID uuid.UUID) (int, error) {
	var number int
	err := tx.NewSelect().
		Table("orders.orders").
		Column("number").
		Where("batch_id = ?", batchID).
		OrderExpr("number DESC").
		Limit(1).
		Scan(ctx, &number)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	return number, err
}
