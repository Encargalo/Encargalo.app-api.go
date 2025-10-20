package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type orders struct {
	db *bun.DB
}

func NewOrdersRepo(db *bun.DB) ports.OrdersRepo {
	return &orders{db}
}

func (o *orders) CreateOrder(ctx context.Context, order *models.Order) error {

	if err := o.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {

		if _, err := tx.NewInsert().Model(order).Exec(ctx); err != nil {
			fmt.Println("error al insertar order: %w", err)
			return err
		}

		if _, err := tx.NewInsert().Model(&order.ItemsOrders).Exec(ctx); err != nil {
			fmt.Println("error al insertar items_orders: %w", err)
			return err
		}

		for _, v := range order.ItemsOrders {

			if len(v.Additions) != 0 {
				if _, err := tx.NewInsert().Model(&v.Additions).Exec(ctx); err != nil {
					fmt.Println("error al insertar el additions_order")
					return err
				}
			}
		}

		return nil

	}); err != nil {

		return err

	}

	return nil
}

func (o *orders) SearchOrdersByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {

	order := new(models.Order)

	if err := o.db.NewSelect().Model(order).Where("id = ?", orderID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		log.Printf("SearchOrderByID: query failed for orderID=%s: %v", orderID, err)
		return nil, errors.New("unexpected error")
	}

	return order, nil

}

func (o *orders) SearchItemsByID(ctx context.Context, ItemsID []uuid.UUID) ([]models.Items, error) {

	var items []models.Items

	err := o.db.NewSelect().Model(&items).Where("id in (?)", bun.In(ItemsID)).Scan(ctx, &items)
	if err != nil {
		fmt.Println(err)
	}

	uniqueIDs := make(map[uuid.UUID]struct{})
	for _, id := range ItemsID {
		uniqueIDs[id] = struct{}{}
	}

	if len(items) != len(uniqueIDs) {
		return nil, fmt.Errorf("one or more items not found")
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
		return nil, fmt.Errorf("one or more additions not found")
	}

	return additions, err

}
