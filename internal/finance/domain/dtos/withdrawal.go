package dtos

// WithdrawalItem es una solicitud en el listado del CRM. AmountBs es lo
// solicitado y PayableBs lo que hay que transferir hoy: son distintos cuando
// el rider cobró efectivo después de pedir el retiro, y el que manda es el
// segundo. Mostrar solo uno de los dos es como se produce un descuadre.
type WithdrawalItem struct {
	ID         string  `json:"id"`
	EntityID   string  `json:"entity_id"`
	Name       string  `json:"name"`
	SurName    string  `json:"sur_name,omitempty"`
	Phone      string  `json:"phone,omitempty"`
	AmountBs   float64 `json:"amount_bs"`
	AmountUSD  float64 `json:"amount_usd"`
	PayableBs  float64 `json:"payable_bs"`
	PayableUSD float64 `json:"payable_usd"`
	// PayableFrozen indica si PayableBs ya está congelado (solicitud
	// reclamada) o si todavía se recalcula en cada consulta.
	PayableFrozen   bool    `json:"payable_frozen"`
	BcvRate         float64 `json:"bcv_rate,omitempty"`
	Status          string  `json:"status"`
	PMPhone         string  `json:"pm_phone,omitempty"`
	PMBankCode      string  `json:"pm_bank_code,omitempty"`
	PMNationalID    string  `json:"pm_national_id,omitempty"`
	ProofURL        *string `json:"proof_url,omitempty"`
	StaffID         *string `json:"staff_id,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	CreatedAt       string  `json:"created_at"`
	ProcessingAt    *string `json:"processing_at,omitempty"`
	CompletedAt     *string `json:"completed_at,omitempty"`
	ResolvedAt      *string `json:"resolved_at,omitempty"`
}

// Total es el total que matchea el filtro, no el tamaño de la página.
type WithdrawalsResponse struct {
	Items []WithdrawalItem `json:"items"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
}

// ClaimResponse devuelve el monto congelado: es el valor exacto que el
// operador debe transferir.
type ClaimResponse struct {
	ID           string  `json:"id"`
	PayableBs    float64 `json:"payable_bs"`
	PayableUSD   float64 `json:"payable_usd"`
	BcvRate      float64 `json:"bcv_rate,omitempty"`
	Status       string  `json:"status"`
	PMPhone      string  `json:"pm_phone,omitempty"`
	PMBankCode   string  `json:"pm_bank_code,omitempty"`
	PMNationalID string  `json:"pm_national_id,omitempty"`
	ClaimedAt    string  `json:"claimed_at"`
}

type CompleteWithdrawalRequest struct {
	ProofImage string `json:"proof_image" validate:"required"`
	Notes      string `json:"notes"`
}

type RejectWithdrawalRequest struct {
	Reason string `json:"reason" validate:"required"`
}
