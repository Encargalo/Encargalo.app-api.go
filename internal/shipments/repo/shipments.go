package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/pkg/id"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
)

type shipmentsRepo struct {
	db *bun.DB
}

const pgUniqueViolation = "23505"

func NewShipmentsRepo(db *bun.DB) ports.ShipmentsRepo {
	return &shipmentsRepo{db: db}
}

func (r *shipmentsRepo) CreateShipment(ctx context.Context, shipment *models.Shipment) error {
	shipment.ID = id.Ensure(shipment.ID)
	_, err := r.db.NewInsert().Model(shipment).Exec(ctx)
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.Field('C') == pgUniqueViolation {
		return errcustom.ErrShipmentAlreadyExists
	}
	if err != nil {
		slog.Error("shipments CreateShipment failed", "customerID", shipment.CustomerID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *shipmentsRepo) GetShipmentByID(ctx context.Context, shipmentID uuid.UUID) (*models.Shipment, error) {
	var shipment models.Shipment
	err := r.db.NewSelect().
		Model(&shipment).
		Where("id = ?", shipmentID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("shipments GetShipmentByID failed", "shipmentID", shipmentID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return &shipment, nil
}

func (r *shipmentsRepo) GetShipmentHistoryByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Shipment, error) {
	var shipments []models.Shipment
	err := r.db.NewSelect().
		Model(&shipments).
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("shipments GetShipmentHistoryByCustomer failed", "customerID", customerID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return shipments, nil
}

func (r *shipmentsRepo) UpdateShipmentStatus(ctx context.Context, shipmentID uuid.UUID, fromStatus, toStatus models.ShipmentStatus) (bool, error) {
	q := r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("status = ?", toStatus).
		Set("updated_at = now()")

	switch toStatus {
	case models.ShipmentStatusAccepted:
		q = q.Set("accepted_at = now()")
	case models.ShipmentStatusDelivered:
		q = q.Set("delivered_at = now()")
	case models.ShipmentStatusCancelled:
		q = q.Set("cancelled_at = now()")
	}

	res, err := q.
		Where("id = ? AND status = ?", shipmentID, fromStatus).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments UpdateShipmentStatus failed", "shipmentID", shipmentID, "fromStatus", fromStatus, "toStatus", toStatus, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("shipments UpdateShipmentStatus: failed to read rows affected", "shipmentID", shipmentID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *shipmentsRepo) SaveShipmentPayment(ctx context.Context, shipmentID uuid.UUID, payment *models.ShipmentPayment) error {
	_, err := r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("payment = ?", payment).
		Set("updated_at = now()").
		Where("id = ?", shipmentID).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments SaveShipmentPayment failed", "shipmentID", shipmentID, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

// SaveShipmentCancelReason solo escribe si el envío no tiene motivo: el
// primero en llegar es la causa real y los avisos posteriores no deben pisarlo.
func (r *shipmentsRepo) SaveShipmentCancelReason(ctx context.Context, shipmentID uuid.UUID, reason models.CancelReason) error {
	_, err := r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("cancel_reason = ?", string(reason)).
		Set("updated_at = now()").
		Where("id = ? AND cancel_reason IS NULL", shipmentID).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments SaveShipmentCancelReason failed", "shipmentID", shipmentID, "reason", reason, "error", err)
		return errcustom.Unexpected(err)
	}
	return nil
}

func (r *shipmentsRepo) ListActiveShipmentsByCustomer(ctx context.Context, customerID uuid.UUID) ([]models.Shipment, error) {
	var shipments []models.Shipment
	err := r.db.NewSelect().
		Model(&shipments).
		Where("customer_id = ?", customerID).
		Where("status IN (?)", bun.In([]models.ShipmentStatus{
			models.ShipmentStatusPaymentReview,
			models.ShipmentStatusSearching,
			models.ShipmentStatusAccepted,
			models.ShipmentStatusInTransit,
		})).
		OrderExpr("requested_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("shipments ListActiveShipmentsByCustomer failed", "customerID", customerID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return shipments, nil
}

// AssignShipmentRider es un UPDATE condicional (WHERE status = searching) sin
// chequeo de "rider ocupado": a diferencia de trips, el pool "orders" ya
// permite varios pedidos/envíos simultáneos por rider (decisión de producto).
func (r *shipmentsRepo) AssignShipmentRider(ctx context.Context, shipmentID, riderID uuid.UUID) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("rider_id = ?", riderID).
		Set("status = ?", models.ShipmentStatusAccepted).
		Set("accepted_at = now()").
		Set("updated_at = now()").
		Where("id = ? AND status = ?", shipmentID, models.ShipmentStatusSearching).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments AssignShipmentRider failed", "shipmentID", shipmentID, "riderID", riderID, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("shipments AssignShipmentRider: failed to read rows affected", "shipmentID", shipmentID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

// AdvanceShipmentStatus colapsa "recogido" en el mismo paso que "en camino"
// (decisión de producto): accepted->in_transit valida pickup_code,
// in_transit->delivered valida dropoff_code. Un solo UPDATE condicional por
// transición, ambos CAS atómicos.
func (r *shipmentsRepo) AdvanceShipmentStatus(ctx context.Context, shipmentID, riderID uuid.UUID, code string) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("status = ?", models.ShipmentStatusInTransit).
		Set("pickup_code_used_at = now()").
		Set("updated_at = now()").
		Where("id = ? AND rider_id = ? AND status = ? AND pickup_code = ? AND pickup_code_used_at IS NULL",
			shipmentID, riderID, models.ShipmentStatusAccepted, code).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments AdvanceShipmentStatus (pickup) failed", "shipmentID", shipmentID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	if rows, rowsErr := res.RowsAffected(); rowsErr == nil && rows > 0 {
		return true, nil
	} else if rowsErr != nil {
		slog.Error("shipments AdvanceShipmentStatus: failed to read rows affected", "shipmentID", shipmentID, "error", rowsErr)
		return false, errcustom.Unexpected(rowsErr)
	}

	res, err = r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("status = ?", models.ShipmentStatusDelivered).
		Set("dropoff_code_used_at = now()").
		Set("delivered_at = now()").
		Set("updated_at = now()").
		Where("id = ? AND rider_id = ? AND status = ? AND dropoff_code = ? AND dropoff_code_used_at IS NULL",
			shipmentID, riderID, models.ShipmentStatusInTransit, code).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments AdvanceShipmentStatus (dropoff) failed", "shipmentID", shipmentID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("shipments AdvanceShipmentStatus: failed to read rows affected", "shipmentID", shipmentID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *shipmentsRepo) ReleaseShipmentRider(ctx context.Context, shipmentID, riderID uuid.UUID) (bool, error) {
	res, err := r.db.NewUpdate().
		TableExpr("shipments.shipments").
		Set("rider_id = NULL").
		Set("status = ?", models.ShipmentStatusSearching).
		Set("accepted_at = NULL").
		Set("updated_at = now()").
		Where("id = ? AND rider_id = ? AND status = ?", shipmentID, riderID, models.ShipmentStatusAccepted).
		Exec(ctx)
	if err != nil {
		slog.Error("shipments ReleaseShipmentRider failed", "shipmentID", shipmentID, "riderID", riderID, "error", err)
		return false, errcustom.Unexpected(err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("shipments ReleaseShipmentRider: failed to read rows affected", "shipmentID", shipmentID, "error", err)
		return false, errcustom.Unexpected(err)
	}
	return rows > 0, nil
}

func (r *shipmentsRepo) ListActiveShipmentsByRider(ctx context.Context, riderID uuid.UUID) ([]models.Shipment, error) {
	var shipments []models.Shipment
	err := r.db.NewSelect().
		Model(&shipments).
		Where("rider_id = ?", riderID).
		Where("status IN (?)", bun.In([]models.ShipmentStatus{
			models.ShipmentStatusAccepted,
			models.ShipmentStatusInTransit,
		})).
		OrderExpr("requested_at DESC").
		Scan(ctx)
	if err != nil {
		slog.Error("shipments ListActiveShipmentsByRider failed", "riderID", riderID, "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return shipments, nil
}

func (r *shipmentsRepo) ListSearchingShipmentIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.NewSelect().
		Model((*models.Shipment)(nil)).
		Column("id").
		Where("status = ?", models.ShipmentStatusSearching).
		Scan(ctx, &ids)
	if err != nil {
		slog.Error("shipments ListSearchingShipmentIDs failed", "error", err)
		return nil, errcustom.Unexpected(err)
	}
	return ids, nil
}
