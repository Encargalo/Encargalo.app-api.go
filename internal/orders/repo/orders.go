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

	err := o.db.NewSelect().Model(&items).Where("id in (?)", bun.In(ItemsID)).Scan(ctx, &items)
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

	var additions []models.Addition

	err := o.db.NewSelect().Model(&additions).Where("id in (?)", bun.In(additionID)).Scan(ctx, &additions)
	if err != nil {
		fmt.Println(err)
	}

	uniqueIDs := make(map[uuid.UUID]struct{})
	for _, id := range additionID {
		uniqueIDs[id] = struct{}{}
	}

	if len(additions) != len(uniqueIDs) {
		return nil, errcustom.ErrItemsNotFound
	}

	return additions, err

}

func (o *orders) SearchActiveOrdersByCriteria(ctx context.Context, criteria uuid.UUID) ([]models.Order, error) {

	var orders []models.Order

	err := o.db.NewSelect().
		Model(&orders).
		Where("status IN (?)", bun.In([]string{
			"Pending",
			"Waiting For Payment Proof",
			"In Preparation",
			"On The Way",
		})).
		Where("(shop_id = ? OR customer_id = ?)", criteria, criteria).
		Scan(ctx)

	if err != nil {
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

	if status == "Cancelado" {
		q = q.Set("canceled_at = ?", time.Now())
	}

	if _, err := q.Exec(ctx); err != nil {
		slog.Error("UpdateOrderStatus: failed to update order status",
			"orderID", orderID, "error", err)
		return err
	}

	return nil
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
