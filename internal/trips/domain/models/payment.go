package models

import (
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"github.com/google/uuid"
)

// TripPayment congela el crédito a la cartera cuando un Pago Móvil aprobado
// no se convierte en viaje. Solo lo ve el cliente (ToCustomerDTO).
type TripPayment struct {
	ExpectedAmountBs    float64    `json:"expected_amount_bs,omitempty"`
	TransactionID       *uuid.UUID `json:"transaction_id,omitempty"`
	CreditedBs          *float64   `json:"credited_bs,omitempty"`
	RefundablePMBs      *float64   `json:"refundable_pm_bs,omitempty"`
	CanRequestRefund    bool       `json:"can_request_refund,omitempty"`
	RefundEligibleUntil *string    `json:"refund_eligible_until,omitempty"`
}

// PagoMovilChargeBs es la tarifa en Bs congelada (fare × bcv_rate) de un viaje
// que no se pagó con saldo y que todavía no tiene crédito de reembolso.
func (t *Trip) PagoMovilChargeBs() (amount, rate float64, ok bool) {
	if t == nil || (t.BalancePaidBs != nil && *t.BalancePaidBs > 0) {
		return 0, 0, false
	}
	if t.Payment != nil && t.Payment.TransactionID != nil {
		return 0, 0, false
	}
	if t.Fare == nil || t.BcvRate == nil || *t.Fare <= 0 || *t.BcvRate <= 0 {
		return 0, 0, false
	}
	return money.RoundBs(*t.Fare * *t.BcvRate), *t.BcvRate, true
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
		ExpectedAmountBs:    p.ExpectedAmountBs,
		TransactionID:       p.TransactionID,
		CreditedBs:          p.CreditedBs,
		RefundablePMBs:      p.RefundablePMBs,
		CanRequestRefund:    p.CanRequestRefund,
		RefundEligibleUntil: p.RefundEligibleUntil,
	}
}
