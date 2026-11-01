package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	shipmentsApp "Encargalo.app-api.go/internal/shipments/app"
	shipmentsDtos "Encargalo.app-api.go/internal/shipments/domain/dtos"
	shipmentsModels "Encargalo.app-api.go/internal/shipments/domain/models"
	"github.com/google/uuid"
)

// riderShipmentOfferDTO es el envío como lo ve el rider: fare_usd/fare_bs son
// su ganancia (RiderPayUSD, 90%), igual que fare_usd/fare_bs en viajes
// (riderTripOfferDTO), convertida con la tasa congelada en la cotización del
// envío. Sin descuento que absorber: a diferencia de trips, Shipment no tiene DiscountUSD.
func riderShipmentOfferDTO(shipment *shipmentsModels.Shipment) shipmentsDtos.ShipmentResponse {
	frozen := *shipment
	frozen.ApplyFrozenBsFare()
	resp := frozen.ToDTO()
	if shipment.Fare == nil {
		return resp
	}
	usd := money.RoundUSD(riderModels.RiderPayUSD(*shipment.Fare))
	resp.FareUSD = &usd
	if shipment.BcvRate != nil && *shipment.BcvRate > 0 {
		bs := money.RoundBs(riderModels.RiderPayUSD(money.RoundBs(*shipment.Fare * *shipment.BcvRate)))
		resp.FareBs = &bs
	}
	return resp
}

// shipmentOfferTimeout es el único vencimiento del broadcast (sin rondas, a
// diferencia de trips): si nadie acepta en este plazo, el envío pasa a no_driver.
const shipmentOfferTimeout = 10 * time.Minute

type shipmentBroadcast struct {
	Type     string                         `json:"type"`
	Shipment shipmentsDtos.ShipmentResponse `json:"shipment"`
}

// OfferShipment manda new_shipment a todo rider conectado en modo orders, sin
// filtro de vehículo (cualquiera lleva <5kg) — a diferencia de BroadcastNewOrder,
// que sí filtra por tipo de negocio. Nunca lleva pickup_code/dropoff_code.
func (r *appRidersManager) OfferShipment(ctx context.Context, shipment shipmentsModels.Shipment) {
	data, err := json.Marshal(shipmentBroadcast{Type: "new_shipment", Shipment: riderShipmentOfferDTO(&shipment)})
	if err != nil {
		slog.Error("error marshaling new_shipment broadcast", "shipmentID", shipment.ID, "error", err)
		return
	}

	r.mu.RLock()
	clients := make([]ridersPorts.RiderConn, 0, len(r.availableRiders))
	riderIDs := make([]uuid.UUID, 0, len(r.availableRiders))
	seen := make(map[uuid.UUID]struct{}, len(r.availableRiders))
	for riderID, c := range r.availableRiders {
		if r.riderMode[riderID] == riderModels.RiderModeTrips {
			continue
		}
		if _, paused := r.pausedRiders[riderID]; paused {
			// Pausado pero conectado: no recibe el broadcast, y tampoco el
			// fallback de push por última ubicación conocida más abajo.
			seen[riderID] = struct{}{}
			continue
		}
		clients = append(clients, c)
		riderIDs = append(riderIDs, riderID)
		seen[riderID] = struct{}{}
	}
	r.mu.RUnlock()

	for _, c := range clients {
		c.Send(data)
	}

	r.pushNewShipment(shipment, riderIDs, seen)
	r.scheduleShipmentOfferTimeout(ctx, shipment.ID)
}

// pushNewShipment manda el push a los riders conectados en modo orders (mismo
// pool que ya recibe el new_shipment por WS) más el respaldo de última
// ubicación conocida alrededor del punto de recogida, para riders sin
// presencia viva — igual que pushNewOrder, sin filtro de vehículo (ver
// OfferShipment: cualquiera lleva <5kg).
func (r *appRidersManager) pushNewShipment(shipment shipmentsModels.Shipment, riderIDs []uuid.UUID, exclude map[uuid.UUID]struct{}) {
	if r.push == nil {
		return
	}
	go func() {
		if r.lastKnown != nil {
			queryCtx, cancel := context.WithTimeout(context.Background(), lastKnownLocationTimeout)
			entries, err := r.lastKnown.SearchNearby(queryCtx, riderModels.LastKnownLocationQuery{
				Latitude:  shipment.PickupLatitude,
				Longitude: shipment.PickupLongitude,
				RadiusKm:  offlineOrderFallbackRadiusKm,
				Mode:      riderModels.RiderModeOrders,
				MaxAge:    riderModels.LastKnownLocationMaxAge,
				Exclude:   exclude,
			})
			cancel()
			if err != nil {
				slog.Error("error buscando riders por última ubicación conocida para new_shipment", "shipmentID", shipment.ID, "error", err)
			} else {
				for _, e := range entries {
					riderIDs = append(riderIDs, e.RiderID)
				}
			}
		}
		if len(riderIDs) == 0 {
			return
		}
		r.push.NotifyNewShipment(context.Background(), riderIDs, shipment)
	}()
}

func (r *appRidersManager) scheduleShipmentOfferTimeout(ctx context.Context, shipmentID uuid.UUID) {
	if err := r.shipmentOfferTimeouts.Schedule(ctx, shipmentID, time.Now().Add(shipmentOfferTimeout)); err != nil {
		slog.Error("error scheduling shipment offer timeout", "shipmentID", shipmentID, "error", err)
	}
}

// onShipmentOfferTimeout corre cuando el ShipmentOfferTimeoutWorker reclama un
// vencimiento cumplido. Sin rondas: si sigue en searching, va directo a no_driver.
func (r *appRidersManager) onShipmentOfferTimeout(ctx context.Context, shipmentID uuid.UUID) {
	current, err := r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		slog.Error("error reading shipment on offer timeout", "shipmentID", shipmentID, "error", err)
		return
	}
	if current.Status != shipmentsModels.ShipmentStatusSearching {
		return
	}
	r.expireShipmentIfStillSearching(shipmentID)
}

// expireShipmentIfStillSearching pasa el envío a no_driver y reembolsa o
// acredita según el método de pago, igual que expireTripIfStillSearching.
func (r *appRidersManager) expireShipmentIfStillSearching(shipmentID uuid.UUID) {
	ctx := context.Background()
	ok, err := r.shipmentsRepo.UpdateShipmentStatus(ctx, shipmentID, shipmentsModels.ShipmentStatusSearching, shipmentsModels.ShipmentStatusNoDriver)
	if err != nil {
		slog.Error("error expiring shipment to no_driver", "shipmentID", shipmentID, "error", err)
	}
	if err != nil || !ok {
		return
	}
	if reasonErr := r.shipmentsRepo.SaveShipmentCancelReason(ctx, shipmentID, shipmentsModels.CancelReasonNoDriver); reasonErr != nil {
		slog.Error("save shipment cancel reason", "shipmentID", shipmentID, "reason", shipmentsModels.CancelReasonNoDriver, "error", reasonErr)
	}
	if r.customersShipment == nil && r.balance == nil {
		return
	}
	shipment, getErr := r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if getErr != nil {
		slog.Error("error loading expired shipment", "shipmentID", shipmentID, "error", getErr)
		return
	}
	if shipment.BalancePaidBs != nil && *shipment.BalancePaidBs > 0 && r.balance != nil {
		if refundErr := r.balance.RefundBalancePayment(ctx, shipment.CustomerID, shipment.ID, shipmentsModels.ReferenceTypeShipment, *shipment.BalancePaidBs); refundErr != nil {
			slog.Error("refund balance on shipment no_driver", "shipmentID", shipment.ID, "error", refundErr)
		}
	}
	shipmentsApp.CreditUnservedPagoMovil(ctx, r.balance, shipment, r.shipmentsRepo.SaveShipmentPayment)
	shipment.SetCancelReason(shipmentsModels.CancelReasonNoDriver)
	r.notifyShipmentCustomer(ctx, shipment)
	r.publishShipmentNoDriver(ctx, shipment)
}

// AcceptShipment asigna por CAS en BD, sin candado en memoria: el pool orders
// ya permite varias asignaciones simultáneas por rider (ver AssignShipmentRider).
func (r *appRidersManager) AcceptShipment(ctx context.Context, riderID, shipmentID uuid.UUID, conn ridersPorts.RiderConn) error {
	shipment, err := r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.Status != shipmentsModels.ShipmentStatusSearching {
		return errcustom.ErrInvalidTransition
	}

	ok, err := r.shipmentsRepo.AssignShipmentRider(ctx, shipmentID, riderID)
	if err != nil {
		return err
	}
	if !ok {
		msg, _ := json.Marshal(map[string]string{"type": "shipment_taken", "shipment_id": shipmentID.String()})
		conn.Send(msg)
		return nil
	}

	if err := r.shipmentOfferTimeouts.Cancel(ctx, shipmentID); err != nil {
		slog.Error("error cancelling shipment offer timeout", "shipmentID", shipmentID, "error", err)
	}

	shipment, err = r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	r.publishShipmentStatusChanged(ctx, shipment)
	accepted, _ := json.Marshal(shipmentBroadcast{Type: "shipment_accepted", Shipment: riderShipmentOfferDTO(shipment)})
	conn.Send(accepted)
	r.notifyShipmentCustomer(ctx, shipment)

	r.mu.Lock()
	r.assignedShipments[riderID] = append(r.assignedShipments[riderID], shipmentID)
	r.shipmentCustomer[shipmentID] = shipment.CustomerID
	loc, hasLoc := r.riderLocation[riderID]
	notifier := r.customersShipment
	r.mu.Unlock()

	// El pin del rider aparece en el cliente apenas acepta, igual que en AcceptOrder.
	if hasLoc && notifier != nil {
		notifier.NotifyRiderLocation(ctx, shipment.CustomerID, shipmentID, loc.Latitude, loc.Longitude)
	}

	return nil
}

// AdvanceShipment mueve accepted→in_transit (valida pickup_code) o delega en
// completeShipment para in_transit→delivered (valida dropoff_code y liquida
// la ganancia). "Recogido" no es un estado propio: ver decisión de producto
// en internal/shipments/domain/models/shipment.go.
func (r *appRidersManager) AdvanceShipment(ctx context.Context, riderID, shipmentID uuid.UUID, code string, conn ridersPorts.RiderConn) error {
	shipment, err := r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.RiderID == nil || *shipment.RiderID != riderID {
		return errcustom.ErrInvalidTransition
	}

	code = strings.TrimSpace(code)
	if shipment.Status == shipmentsModels.ShipmentStatusInTransit {
		return r.completeShipment(ctx, shipment, code, conn)
	}
	if shipment.Status != shipmentsModels.ShipmentStatusAccepted {
		return errcustom.ErrInvalidTransition
	}

	updated, err := r.shipmentsRepo.AdvanceShipmentStatus(ctx, shipmentID, riderID, code)
	if err != nil {
		return err
	}
	if !updated {
		return errcustom.ErrInvalidCode
	}

	shipment, err = r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	r.publishShipmentStatusChanged(ctx, shipment)
	data, _ := json.Marshal(shipmentBroadcast{Type: "shipment_status", Shipment: riderShipmentOfferDTO(shipment)})
	conn.Send(data)
	r.notifyShipmentCustomer(ctx, shipment)

	return nil
}

// completeShipment usa el fare que ya quedó fijado en shipment.Fare desde
// RequestShipment, no lo recalcula — igual que completeTrip.
func (r *appRidersManager) completeShipment(ctx context.Context, shipment *shipmentsModels.Shipment, dropoffCode string, conn ridersPorts.RiderConn) error {
	if !r.beginCompletingShipment(shipment.ID) {
		return errcustom.ErrInvalidTransition
	}
	defer r.endCompletingShipment(shipment.ID)

	if dropoffCode == "" {
		return errcustom.ErrInvalidCode
	}
	if shipment.Fare == nil || shipment.RiderID == nil {
		return errcustom.ErrUnexpectedError
	}
	fare := *shipment.Fare
	riderID := *shipment.RiderID

	updated, err := r.shipmentsRepo.AdvanceShipmentStatus(ctx, shipment.ID, riderID, dropoffCode)
	if err != nil {
		return err
	}
	if !updated {
		return errcustom.ErrInvalidCode
	}

	delivered, err := r.shipmentsRepo.GetShipmentByID(ctx, shipment.ID)
	if err != nil {
		return err
	}
	// ApplyFrozenBsFare es para la vista del cliente (fare completo); la del
	// rider la recalcula riderShipmentOfferDTO sobre su propia copia.
	delivered.ApplyFrozenBsFare()
	r.publishShipmentStatusChanged(ctx, delivered)
	data, _ := json.Marshal(shipmentBroadcast{Type: "shipment_status", Shipment: riderShipmentOfferDTO(delivered)})
	conn.Send(data)
	r.notifyShipmentCustomer(ctx, delivered)

	r.mu.Lock()
	r.forgetShipmentLocked(shipment.ID)
	r.mu.Unlock()

	if earnErr := r.shipmentEarnings.RecordShipmentEarning(ctx, riderID, shipment.ID, fare, r.bcvRate(ctx)); earnErr != nil {
		slog.Error("error recording shipment earning", "shipmentID", shipment.ID, "riderID", riderID, "error", earnErr)
	}

	return nil
}

func (r *appRidersManager) beginCompletingShipment(shipmentID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, inFlight := r.completingShipments[shipmentID]; inFlight {
		return false
	}
	r.completingShipments[shipmentID] = struct{}{}
	return true
}

func (r *appRidersManager) endCompletingShipment(shipmentID uuid.UUID) {
	r.mu.Lock()
	delete(r.completingShipments, shipmentID)
	r.mu.Unlock()
}

// CancelAcceptedShipment libera un envío accepted y lo vuelve a ofertar a todo
// el pool: a diferencia de CancelAcceptedTrip no hay lista de exclusión que
// armar, el broadcast ya es abierto a todos.
func (r *appRidersManager) CancelAcceptedShipment(ctx context.Context, riderID, shipmentID uuid.UUID, conn ridersPorts.RiderConn) error {
	shipment, err := r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	if shipment.RiderID == nil || *shipment.RiderID != riderID {
		return errcustom.ErrInvalidTransition
	}
	if shipment.Status != shipmentsModels.ShipmentStatusAccepted {
		return errcustom.ErrInvalidTransition
	}

	ok, err := r.shipmentsRepo.ReleaseShipmentRider(ctx, shipmentID, riderID)
	if err != nil {
		return err
	}
	if !ok {
		return errcustom.ErrInvalidTransition
	}
	slog.Info("rider cancelled accepted shipment", "shipmentID", shipmentID, "riderID", riderID, "at", time.Now().UTC())

	shipment, err = r.shipmentsRepo.GetShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	r.publishShipmentStatusChanged(ctx, shipment)
	released, _ := json.Marshal(shipmentBroadcast{Type: "shipment_status", Shipment: riderShipmentOfferDTO(shipment)})
	conn.Send(released)
	r.notifyShipmentCustomer(ctx, shipment)

	r.mu.Lock()
	r.forgetShipmentLocked(shipment.ID)
	r.mu.Unlock()

	r.OfferShipment(ctx, *shipment)
	return nil
}

// NotifyShipmentCancelled avisa solo al rider ya asignado, si lo hay: quien
// apenas vio el broadcast no perdió nada que requiera avisarle fuera del WS.
func (r *appRidersManager) NotifyShipmentCancelled(ctx context.Context, shipment shipmentsModels.Shipment) {
	if err := r.shipmentOfferTimeouts.Cancel(ctx, shipment.ID); err != nil {
		slog.Error("error cancelling shipment offer timeout", "shipmentID", shipment.ID, "error", err)
	}

	r.mu.Lock()
	r.forgetShipmentLocked(shipment.ID)
	r.mu.Unlock()

	if shipment.RiderID == nil {
		return
	}

	r.pushShipmentCancelled(shipment, *shipment.RiderID)

	r.mu.RLock()
	conn, ok := r.availableRiders[*shipment.RiderID]
	r.mu.RUnlock()
	if !ok {
		return
	}
	payload, err := json.Marshal(map[string]string{"type": "shipment_cancelled", "shipment_id": shipment.ID.String()})
	if err != nil {
		slog.Error("error marshaling shipment_cancelled", "shipmentID", shipment.ID, "error", err)
		return
	}
	conn.Send(payload)
}

// pushShipmentCancelled corre en su propia goroutine por la misma razón que
// pushOrderCancelled: no puede agregar latencia al camino síncrono de la cancelación.
func (r *appRidersManager) pushShipmentCancelled(shipment shipmentsModels.Shipment, riderID uuid.UUID) {
	if r.push == nil {
		return
	}
	go r.push.NotifyShipmentCancelled(context.Background(), []uuid.UUID{riderID}, shipment)
}

func (r *appRidersManager) notifyShipmentCustomer(ctx context.Context, shipment *shipmentsModels.Shipment) {
	if r.customersShipment == nil || shipment == nil {
		return
	}
	r.customersShipment.NotifyShipment(ctx, *shipment)
}

// publishShipmentStatusChanged avisa al ETL que CubeShipment debe
// resincronizarse — mismo motivo que publishTripStatusChanged.
func (r *appRidersManager) publishShipmentStatusChanged(ctx context.Context, shipment *shipmentsModels.Shipment) {
	if r.stream == nil {
		return
	}
	event := shipmentsModels.DataEventShipment{}
	event.BuildFromShipment(shipmentsModels.EventShipmentStatusChanged, *shipment)
	if err := r.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing shipment_status_changed event", "shipmentID", shipment.ID, "error", err)
	}
}

func (r *appRidersManager) publishShipmentNoDriver(ctx context.Context, shipment *shipmentsModels.Shipment) {
	if r.stream == nil {
		return
	}
	event := shipmentsModels.DataEventShipment{}
	event.BuildFromShipment(shipmentsModels.EventShipmentNoDriver, *shipment)
	if err := r.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing shipment_no_driver event", "shipmentID", shipment.ID, "error", err)
	}
}
