package models

import (
	"time"

	"Encargalo.app-api.go/internal/pkg/money"
	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	"github.com/google/uuid"
)

// ShipmentPayment congela lo que el cliente necesita saber de su dinero: el
// estado de la revisión del comprobante y el crédito a la cartera cuando un
// Pago Móvil no se convierte en envío. Solo lo ve el cliente (ToCustomerDTO).
type ShipmentPayment struct {
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
// se cubrió con Pago Móvil, descontando el balance, mientras no tenga crédito
// de reembolso. A diferencia de trips no hay efectivo que restar.
func (s *Shipment) PagoMovilChargeBs() (amount, rate float64, ok bool) {
	if s == nil || s.Fare == nil || s.BcvRate == nil || *s.Fare <= 0 || *s.BcvRate <= 0 {
		return 0, 0, false
	}
	if s.Payment != nil && s.Payment.TransactionID != nil {
		return 0, 0, false
	}
	amount = money.RoundBs(*s.Fare**s.BcvRate - bsOrZero(s.BalancePaidBs))
	if amount <= 0.01 {
		return 0, 0, false
	}
	return amount, *s.BcvRate, true
}

func bsOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// HasVerifiedPagoMovil es true cuando el staff ya aprobó el comprobante
// (el envío salió de payment_review) y el cobro sigue sin acreditar.
func (s *Shipment) HasVerifiedPagoMovil() bool {
	if s == nil {
		return false
	}
	switch s.Status {
	case ShipmentStatusSearching, ShipmentStatusAccepted, ShipmentStatusInTransit:
	default:
		return false
	}
	_, _, ok := s.PagoMovilChargeBs()
	return ok
}

// RecordPagoMovilRefund guarda el crédito. eligibleUntil distinto de nil
// habilita la solicitud de devolución a la cuenta del cliente.
func (s *Shipment) RecordPagoMovilRefund(txID uuid.UUID, amount float64, eligibleUntil *time.Time) {
	if s == nil || txID == uuid.Nil {
		return
	}
	id := txID
	credited := amount
	payment := &ShipmentPayment{
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
	s.Payment = payment
}

func (p *ShipmentPayment) toDTO() *dtos.ShipmentPayment {
	if p == nil {
		return nil
	}
	return &dtos.ShipmentPayment{
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
