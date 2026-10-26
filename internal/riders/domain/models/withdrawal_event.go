package models

import financeModels "Encargalo.app-api.go/internal/finance/domain/models"

// DataEventWithdrawalRequested avisa al equipo que hay un retiro esperando
// pago, para que no dependa de que alguien esté mirando el CRM.
type DataEventWithdrawalRequested struct {
	MessageType string  `json:"type"`
	RequestID   string  `json:"request_id"`
	RiderID     string  `json:"rider_id"`
	AmountBs    float64 `json:"amount_bs"`
	PMPhone     string  `json:"pm_phone"`
}

func (d *DataEventWithdrawalRequested) BuildFrom(req financeModels.WithdrawalRequest) {
	d.MessageType = "withdrawal_requested"
	d.RequestID = req.ID.String()
	d.RiderID = req.EntityID.String()
	d.AmountBs = req.Amount
	if req.PMPhone != nil {
		d.PMPhone = *req.PMPhone
	}
}
