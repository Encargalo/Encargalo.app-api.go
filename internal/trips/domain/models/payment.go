package models

import (
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
)

// TripPayment congela lo que el cliente necesita saber de su dinero: el estado
// de la revisión del comprobante y el crédito a la cartera cuando un Pago Móvil
// no se convierte en viaje. Solo lo ve el cliente (ToCustomerDTO).
type TripPayment struct {
	Status              string     `json:"status,omitempty"`
	ExpectedAmountBs    float64    `json:"expected_amount_bs,omitempty"`
	ObservedAmountBs    *float64   `json:"observed_amount_bs,omitempty"`
	DifferenceBs        *float64   `json:"difference_bs,omitempty"`
	RejectionReason     *string    `json:"rejection_reason,omitempty"`
	TransactionID       *uuid.UUID `json:"transaction_id,omitempty"`
	CreditedBs          *float64   `json:"credited_bs,omitempty"`
	RefundablePMBs      *float64   `json:"refundable_pm_bs,omitempty"`
	CanRequestRefund    bool       `json:"can_request_refund,omitempty"`
	RefundEligibleUntil *string    `json:"refund_eligible_until,omitempty"`
}

// PagoMovilChargeBs es la porción de la tarifa congelada (fare × bcv_rate) que
// se cubrió con Pago Móvil, descontando saldo y efectivo, mientras no tenga
// crédito de reembolso.
func (t *Trip) PagoMovilChargeBs() (amount, rate float64, ok bool) {
	if t == nil || t.Fare == nil || t.BcvRate == nil || *t.Fare <= 0 || *t.BcvRate <= 0 {
		return 0, 0, false
	}
	if t.Payment != nil && t.Payment.TransactionID != nil {
		return 0, 0, false
	}
	// Efectivo puro no guarda cash_amount_bs: toda la tarifa se paga al rider.
	if t.MethodPayment == MethodEfectivo && t.CashAmountBs == nil {
		return 0, 0, false
	}
	amount = money.RoundBs(*t.Fare**t.BcvRate - bsOrZero(t.BalancePaidBs) - bsOrZero(t.CashAmountBs))
	if amount <= 0.01 {
		return 0, 0, false
	}
	return amount, *t.BcvRate, true
}

func bsOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// HasVerifiedPagoMovil es true cuando el staff ya aprobó el comprobante
// (el viaje salió de payment_review) y el cobro sigue sin acreditar.
func (t *Trip) HasVerifiedPagoMovil() bool {
	if t == nil {
		return false
	}
	switch t.Status {
	case TripStatusSearching, TripStatusAccepted, TripStatusArriving:
	default:
		return false
	}
	_, _, ok := t.PagoMovilChargeBs()
	return ok
}

// RecordPagoMovilRefund guarda el crédito. eligibleUntil distinto de nil
// habilita la solicitud de devolución a la cuenta del cliente.
func (t *Trip) RecordPagoMovilRefund(txID uuid.UUID, amount float64, eligibleUntil *time.Time) {
	if t == nil || txID == uuid.Nil {
		return
	}
	id := txID
	credited := amount
	payment := &TripPayment{
		ExpectedAmountBs: amount,
		TransactionID:    &id,
		CreditedBs:       &credited,
	}
	if eligibleUntil != nil {
		refundable := amount
		until := eligibleUntil.Format(time.RFC3339)
		payment.RefundablePMBs = &refundable
		payment.RefundEligibleUntil = &until
		if !time.Now().After(*eligibleUntil) {
			payment.CanRequestRefund = true
		}
	}
	t.Payment = payment
}

func (p *TripPayment) toDTO() *dtos.TripPayment {
	if p == nil {
		return nil
	}
	return &dtos.TripPayment{
		Status:              p.Status,
		ExpectedAmountBs:    p.ExpectedAmountBs,
		ObservedAmountBs:    p.ObservedAmountBs,
		DifferenceBs:        p.DifferenceBs,
		RejectionReason:     p.RejectionReason,
		TransactionID:       p.TransactionID,
		CreditedBs:          p.CreditedBs,
		RefundablePMBs:      p.RefundablePMBs,
		CanRequestRefund:    p.CanRequestRefund,
		RefundEligibleUntil: p.RefundEligibleUntil,
	}
}
