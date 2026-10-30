package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/id"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	financeModels "Encargalo.app-api.go/internal/finance/domain/models"
	financePorts "Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/tripfare"
	"github.com/google/uuid"
)

const entityTypeRider = "rider"

type financeApp struct {
	repo        ports.FinanceRepo
	bcv         bcvPorts.BcvApp
	withdrawals ports.WithdrawalStore
	payments    ports.PaymentDetailsStore
	events      ports.WithdrawalPublisher
}

func NewFinanceApp(
	repo ports.FinanceRepo,
	bcv bcvPorts.BcvApp,
	withdrawals ports.WithdrawalStore,
	payments ports.PaymentDetailsStore,
	events ports.WithdrawalPublisher,
) ports.FinanceApp {
	return &financeApp{repo: repo, bcv: bcv, withdrawals: withdrawals, payments: payments, events: events}
}

// RecordRideEarning liquida el domicilio al completar. En Pago Móvil acredita
// el 90% del delivery fee: la plataforma tiene el dinero y le debe su pago.
// En efectivo debita lo que cobró y no es suyo (total − pago). bcvRate es la
// tasa que el caller resolvió para este movimiento (ver orders/app al
// completar la entrega). Falla en vez de mover Bs=0 si no hay tasa.
func (f *financeApp) RecordRideEarning(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, totalPrice, bcvRate float64, method string) error {
	if bcvRate <= 0 {
		return errcustom.ErrBcvRateUnavailable
	}
	rate := bcvRate
	serviceType := models.ServiceTypeDelivery
	movement := models.MovementTypeRideBank
	deltaUSD := models.RiderPayUSD(deliveryFee)
	if method == models.PaymentMethodEfectivo {
		// Ya cobró el total. Su pago se queda en el bolsillo; el balance baja
		// solo por el resto (lo del local y la comisión).
		movement = models.MovementTypeRideCash
		deltaUSD = -models.CashOwedUSD(totalPrice, models.RiderPayUSD(deliveryFee))
	}
	deltaBs := money.RoundBs(deltaUSD * bcvRate)

	tx := models.Transaction{
		ID:            id.New(),
		OrderID:       &orderID,
		MovementType:  string(movement),
		ServiceType:   &serviceType,
		PaymentMethod: &method,
		Currency:      models.CurrencyVES,
		BcvRate:       &rate,
		CreatedAt:     time.Now(),
	}

	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, deltaBs, 0, tx)
	return err
}

// RecordRideCashCollected mueve el balance por el pago del rider menos el
// efectivo que ya cobró. Si el efectivo es mayor, el delta es negativo.
func (f *financeApp) RecordRideCashCollected(ctx context.Context, riderID uuid.UUID, orderID uuid.UUID, deliveryFee, cashUSD, bcvRate float64) error {
	if bcvRate <= 0 {
		return errcustom.ErrBcvRateUnavailable
	}
	deltaBs := money.RoundBs((models.RiderPayUSD(deliveryFee) - cashUSD) * bcvRate)
	rate := bcvRate
	serviceType := models.ServiceTypeDelivery
	method := models.PaymentMethodEfectivo
	tx := models.Transaction{
		ID:            id.New(),
		OrderID:       &orderID,
		MovementType:  string(models.MovementTypeRideCash),
		ServiceType:   &serviceType,
		PaymentMethod: &method,
		Currency:      models.CurrencyVES,
		BcvRate:       &rate,
		CreatedAt:     time.Now(),
	}
	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, deltaBs, 0, tx)
	return err
}

// RecordTripEarning credits the rider's balance with RiderShare of the trip
// fare (USD, calculado por distancia vial pickup→dropoff) más el bono de
// acercamiento al pickup (tripfare.PickupBonus), convertido a Bs con bcvRate
// — análogo a RecordRideEarning pero sin payment_method (los viajes no lo
// tienen todavía).
func (f *financeApp) RecordTripEarning(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, pickupDistanceKm, bcvRate float64) error {
	if bcvRate <= 0 {
		return errcustom.ErrBcvRateUnavailable
	}
	bonus := tripfare.PickupBonus(pickupDistanceKm)
	deltaBs := money.RoundBs(models.RiderPayUSD(fare+bonus) * bcvRate)
	rate := bcvRate
	serviceType := models.ServiceTypeTrip

	tx := models.Transaction{
		ID:           id.New(),
		TripID:       &tripID,
		MovementType: string(models.MovementTypeTripBank),
		ServiceType:  &serviceType,
		Currency:     models.CurrencyVES,
		BcvRate:      &rate,
		CreatedAt:    time.Now(),
	}
	// Solo se deja rastro de la distancia en el ledger cuando de verdad disparó un bono.
	if bonus > 0 {
		tx.DistanceKm = &pickupDistanceKm
	}

	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, deltaBs, 0, tx)
	return err
}

// RecordTripCash debita la parte de la tarifa que el rider cobró y no es su
// pago (10%). El 90% ya lo tiene en el efectivo del pasajero.
func (f *financeApp) RecordTripCash(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, bcvRate float64) error {
	if bcvRate <= 0 {
		return errcustom.ErrBcvRateUnavailable
	}
	deltaBs := money.RoundBs(-models.CashOwedUSD(fare, models.RiderPayUSD(fare)) * bcvRate)
	rate := bcvRate
	serviceType := models.ServiceTypeTrip
	method := models.PaymentMethodEfectivo

	tx := models.Transaction{
		ID:            id.New(),
		TripID:        &tripID,
		MovementType:  string(models.MovementTypeTripCash),
		ServiceType:   &serviceType,
		PaymentMethod: &method,
		Currency:      models.CurrencyVES,
		BcvRate:       &rate,
		CreatedAt:     time.Now(),
	}

	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, deltaBs, 0, tx)
	return err
}

// RecordTripCashCollected mueve el balance por el pago del rider menos el
// efectivo que cobró de la tarifa. El resto de la tarifa ya lo tiene la plataforma.
func (f *financeApp) RecordTripCashCollected(ctx context.Context, riderID uuid.UUID, tripID uuid.UUID, fare, cashUSD, bcvRate float64) error {
	if bcvRate <= 0 {
		return errcustom.ErrBcvRateUnavailable
	}
	deltaBs := money.RoundBs((models.RiderPayUSD(fare) - cashUSD) * bcvRate)
	rate := bcvRate
	serviceType := models.ServiceTypeTrip
	method := models.PaymentMethodEfectivo
	tx := models.Transaction{
		ID:            id.New(),
		TripID:        &tripID,
		MovementType:  string(models.MovementTypeTripCash),
		ServiceType:   &serviceType,
		PaymentMethod: &method,
		Currency:      models.CurrencyVES,
		BcvRate:       &rate,
		CreatedAt:     time.Now(),
	}
	_, err := f.repo.RecordMovement(ctx, riderID, entityTypeRider, deltaBs, 0, tx)
	return err
}

func (f *financeApp) GetBalance(ctx context.Context, riderID uuid.UUID) (dtos.BalanceResponse, error) {
	balanceRec, err := f.repo.GetBalance(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.BalanceResponse{}, err
	}
	balance := balanceRec.Amount
	currency := balanceRec.Currency

	txs, err := f.repo.ListRecentTransactions(ctx, riderID, entityTypeRider, 10)
	if err != nil {
		return dtos.BalanceResponse{}, err
	}

	spotRate := f.spotBcvRate(ctx)
	balanceBs, balanceUSD := money.BalanceDual(balance, currency, spotRate)
	minBs := f.withdrawalMinBs(ctx)
	return dtos.BalanceResponse{
		BalanceBs:        balanceBs,
		BalanceUSD:       balanceUSD,
		BcvRate:          spotRate,
		Zone:             getZone(balanceBs, minBs),
		WithdrawalMinBs:  minBs,
		ActiveWithdrawal: f.activeWithdrawal(ctx, riderID, spotRate),
		Transactions:     toTransactionDTOs(txs, spotRate),
	}, nil
}

// activeWithdrawal distingue un balance en 0 por retiro en curso de uno por no
// haber ganado nada. Un fallo acá no debe tumbar la consulta de balance.
func (f *financeApp) activeWithdrawal(ctx context.Context, riderID uuid.UUID, spotRate float64) *dtos.ActiveWithdrawalDTO {
	if f.withdrawals == nil {
		return nil
	}
	req, err := f.withdrawals.GetActive(ctx, riderID, entityTypeRider)
	if err != nil || req == nil {
		return nil
	}
	amountBs, amountUSD := money.BalanceDual(req.Amount, models.CurrencyVES, spotRate)
	return &dtos.ActiveWithdrawalDTO{
		ID:        req.ID.String(),
		AmountBs:  amountBs,
		AmountUSD: amountUSD,
		Status:    string(req.Status),
		CreatedAt: req.CreatedAt.Format(time.RFC3339),
	}
}

// withdrawalMinBs convierte el piso de negocio (WithdrawalMinUSD) a Bs con la
// tasa BCV vigente. Si la tasa no está disponible devuelve 0 (informativo;
// RequestWithdrawal sí falla explícitamente en ese caso, ver abajo).
func (f *financeApp) withdrawalMinBs(ctx context.Context) float64 {
	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return 0
	}
	return money.RoundBs(models.WithdrawalMinUSD * rate.Rate)
}

// RequestWithdrawal crea una solicitud, no un pago: retiene el saldo y deja
// registrada la deuda hasta que staff transfiera y cargue el comprobante. El
// cooldown se mide contra el último retiro pagado, así que un rechazo no lo
// quema.
func (f *financeApp) RequestWithdrawal(ctx context.Context, riderID uuid.UUID) (dtos.WithdrawalResponse, error) {
	lastPaid, err := f.withdrawals.LastCompletedAt(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.WithdrawalResponse{}, err
	}
	if lastPaid != nil && time.Since(*lastPaid) < financeModels.WithdrawalCooldown {
		return dtos.WithdrawalResponse{}, errcustom.ErrWithdrawalCooldown
	}

	active, err := f.withdrawals.GetActive(ctx, riderID, entityTypeRider)
	if err != nil {
		return dtos.WithdrawalResponse{}, err
	}
	if active != nil {
		return dtos.WithdrawalResponse{}, errcustom.ErrWithdrawalAlreadyRequested
	}

	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return dtos.WithdrawalResponse{}, errcustom.ErrBcvRateUnavailable
	}
	minBs := money.RoundBs(models.WithdrawalMinUSD * rate.Rate)
	if minBs <= 0 {
		return dtos.WithdrawalResponse{}, errcustom.ErrBcvRateUnavailable
	}

	pm, err := f.payments.GetPagoMovil(ctx, riderID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return dtos.WithdrawalResponse{}, errcustom.ErrPagoMovilRequired
		}
		return dtos.WithdrawalResponse{}, err
	}

	req, err := f.withdrawals.Create(ctx, financePorts.CreateWithdrawalParams{
		EntityID:     riderID,
		EntityType:   entityTypeRider,
		MinAmount:    minBs,
		Currency:     models.CurrencyVES,
		BcvRate:      rate.Rate,
		PMPhone:      pm.Phone,
		PMBankCode:   pm.BankCode,
		PMNationalID: pm.NationalID,
	})
	if err != nil {
		return dtos.WithdrawalResponse{}, err
	}

	// El aviso es conveniencia, no fuente de verdad: la solicitud ya existe y
	// el CRM la muestra igual si la publicación falla.
	if f.events != nil {
		if pubErr := f.events.PublishWithdrawalRequested(ctx, req); pubErr != nil {
			slog.Error("RequestWithdrawal: publish event failed", "requestID", req.ID, "error", pubErr)
		}
	}

	amountBs, amountUSD := money.BalanceDual(req.Amount, models.CurrencyVES, rate.Rate)
	return dtos.WithdrawalResponse{
		ID:        req.ID.String(),
		AmountBs:  amountBs,
		AmountUSD: amountUSD,
		BcvRate:   rate.Rate,
		Status:    string(req.Status),
		PMPhone:   pm.Phone,
		CreatedAt: req.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (f *financeApp) ListTransactions(ctx context.Context, riderID uuid.UUID, page, limit int) (dtos.TransactionsResponse, error) {
	if limit > 50 {
		limit = 50
	}
	txs, total, err := f.repo.ListTransactions(ctx, riderID, entityTypeRider, page, limit)
	if err != nil {
		return dtos.TransactionsResponse{}, err
	}

	return dtos.TransactionsResponse{
		Transactions: toTransactionDTOs(txs, f.spotBcvRate(ctx)),
		Total:        total,
		Page:         page,
		Limit:        limit,
	}, nil
}

func getZone(balance, withdrawalMinBs float64) models.Zone {
	if withdrawalMinBs > 0 && balance >= withdrawalMinBs {
		return models.ZoneWithdrawalAvailable
	}
	return models.ZoneNormal
}

// RiderEarning returns the rider's net earning from a delivery fee (90%).
func RiderEarning(deliveryFee float64) float64 {
	return deliveryFee * models.RiderShare
}

func (f *financeApp) spotBcvRate(ctx context.Context) float64 {
	if f.bcv == nil {
		return 0
	}
	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return 0
	}
	return rate.Rate
}

func toTransactionDTOs(txs []models.Transaction, spotRate float64) []dtos.TransactionDTO {
	result := make([]dtos.TransactionDTO, 0, len(txs))
	for _, t := range txs {
		txCurrency := money.NormalizeCurrency(t.Currency, models.CurrencyVES)
		amountBs, amountUSD := money.TransactionDual(t.Amount, txCurrency, t.BcvRate, spotRate)
		dto := dtos.TransactionDTO{
			ID:            t.ID.String(),
			AmountBs:      amountBs,
			AmountUSD:     amountUSD,
			BcvRate:       t.BcvRate,
			MovementType:  t.MovementType,
			ServiceType:   t.ServiceType,
			PaymentMethod: t.PaymentMethod,
			DistanceKm:    t.DistanceKm,
			Notes:         t.Notes,
			ProofURL:      t.ProofURL,
			CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		}
		if t.StaffID != nil {
			s := t.StaffID.String()
			dto.StaffID = &s
		}
		if t.OrderID != nil {
			s := t.OrderID.String()
			dto.OrderID = &s
		}
		if t.TripID != nil {
			s := t.TripID.String()
			dto.TripID = &s
		}
		result = append(result, dto)
	}
	return result
}
