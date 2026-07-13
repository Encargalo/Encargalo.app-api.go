package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"

	items "Encargalo.app-api.go/internal/products/domain/models"
)

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
			return err
		}

		order.Number = lastNum + 1

		if _, err := tx.NewInsert().Model(order).Exec(ctx); err != nil {
			slog.Error("CreateOrder: failed to insert order", "orderID", order.ID, "error", err)
			return err
		}

		if _, err := tx.NewInsert().Model(&order.ItemsOrders).Exec(ctx); err != nil {
			slog.Error("CreateOrder: failed to insert items_orders", "orderID", order.ID, "error", err)
			return err
		}

		for _, v := range order.ItemsOrders {
			if len(v.Additions) == 0 {
				continue
			}

			if _, err := tx.NewInsert().Model(&v.Additions).Exec(ctx); err != nil {
				slog.Error("CreateOrder: failed to insert items_orders_additions", "orderID", order.ID, "error", err)
				return err
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
		lastBatch := o.db.NewSelect().
			TableExpr("orders.orders").
			ColumnExpr("batch_id").
			Where("shop_id = ?", criteria).
			OrderExpr("created_at DESC").
			Limit(1)

		q = o.db.NewSelect().
			Model(&orders).
			Where("shop_id = ?", criteria).
			Where("batch_id = (?)", lastBatch)
	} else {
		q = o.db.NewSelect().
			Model(&orders).
			Where("customer_id = ?", criteria).
			Where("status IN (?)", bun.In([]string{
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
		return err
	}

	return nil
}

func (o *orders) SavePickupCode(ctx context.Context, orderID uuid.UUID, code string) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("pickup_code = ?", code).
		Where("id = ?", orderID).
		Exec(ctx)
	return err
}

func (o *orders) SaveDeliveryCode(ctx context.Context, orderID uuid.UUID, code string) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("delivery_code = ?", code).
		Where("id = ?", orderID).
		Exec(ctx)
	return err
}

func (o *orders) MarkPickupCodeUsed(ctx context.Context, orderID uuid.UUID) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("pickup_code_used_at = NOW()").
		Where("id = ?", orderID).
		Exec(ctx)
	return err
}

func (o *orders) MarkDeliveryCodeUsed(ctx context.Context, orderID uuid.UUID) error {
	_, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("delivery_code_used_at = NOW()").
		Where("id = ?", orderID).
		Exec(ctx)
	return err
}

func (o *orders) AssignRider(ctx context.Context, orderID uuid.UUID, riderID uuid.UUID) error {
	res, err := o.db.NewUpdate().Model(&models.Order{}).
		Set("rider_id = ?", riderID).
		Set("rider_accepted_at = NOW()").
		Where("id = ? AND rider_id IS NULL", orderID).
		Exec(ctx)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
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
