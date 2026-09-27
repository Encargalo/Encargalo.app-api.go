package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"time"

	bcvPorts "Encargalo.app-api.go/internal/bcv/domain/ports"
	paymentsModels "Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/pkg/money"
	uploadimage "Encargalo.app-api.go/internal/pkg/uploadImage"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

const refundAmountTolerance = 0.01

type customerFinanceApp struct {
	repo     ports.CustomerFinanceRepo
	hub      ports.StaffRefundHub
	stream   ports.StreamPublisher
	uploader uploadimage.UploadImage
	bcv      bcvPorts.BcvApp
	notifier ports.CustomerNotifier
}

func NewCustomerFinanceApp(
	repo ports.CustomerFinanceRepo,
	hub ports.StaffRefundHub,
	stream ports.StreamPublisher,
	uploader uploadimage.UploadImage,
	bcv bcvPorts.BcvApp,
) ports.CustomerFinanceApp {
	return &customerFinanceApp{repo: repo, hub: hub, stream: stream, uploader: uploader, bcv: bcv}
}

func (f *customerFinanceApp) SetCustomerNotifier(n ports.CustomerNotifier) {
	f.notifier = n
}

func withinRefundWindow(createdAt, now time.Time) bool {
	until := createdAt.Add(models.RefundRequestWindow)
	return !now.After(until)
}

func validateRefundProofURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path == "" {
		return errcustom.ErrInvalidProofURL
	}
	host := strings.ToLower(u.Hostname())
	if host == "res.cloudinary.com" || strings.HasSuffix(host, ".cloudinary.com") {
		return nil
	}
	return errcustom.ErrInvalidProofURL
}

func (f *customerFinanceApp) GetBalance(ctx context.Context, customerID uuid.UUID) (dto.CustomerBalanceResponse, error) {
	balanceRec, err := f.repo.GetBalance(ctx, customerID, models.EntityTypeCustomer)
	if err != nil {
		return dto.CustomerBalanceResponse{}, err
	}
	balance := balanceRec.Amount
	currency := balanceRec.Currency

	txs, _, err := f.repo.ListTransactions(ctx, customerID, models.EntityTypeCustomer, 1, 10)
	if err != nil {
		return dto.CustomerBalanceResponse{}, err
	}

	spotRate := f.spotBcvRate(ctx)
	balanceBs, balanceUSD := money.BalanceDual(balance, currency, spotRate)

	now := time.Now()
	items := make([]dto.CustomerTransactionItem, len(txs))
	for i, t := range txs {
		txCurrency := money.NormalizeCurrency(t.Currency, models.CurrencyVES)
		amountBs, amountUSD := money.TransactionDual(t.Amount, txCurrency, t.BcvRate, spotRate)
		items[i] = dto.CustomerTransactionItem{
			ID:           t.ID.String(),
			AmountBs:     amountBs,
			AmountUSD:    amountUSD,
			MovementType: t.MovementType,
			CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		}
		if orderID := orderIDFromReference(t.ReferenceID, t.ReferenceType); orderID != nil {
			items[i].OrderID = orderID
		}
		if tripID := tripIDFromReference(t.ReferenceID, t.ReferenceType); tripID != nil {
			items[i].TripID = tripID
		}
		if t.RefundRequestID != nil {
			s := t.RefundRequestID.String()
			items[i].RefundRequestID = &s
		}
		if t.RefundablePMBs != nil {
			v := *t.RefundablePMBs
			items[i].RefundablePMBs = &v
		}

		refundableBs := refundableAmount(t)
		rate := 0.0
		if t.BcvRate != nil {
			rate = *t.BcvRate
		}
		// can_request_refund = elegible por ventana/monto/saldo. El débito de la
		// solicitud ocurre al crear; el balance ya refleja holds pending/accepted.
		// El umbral mínimo (más de $1) se compara en Bs con la tasa congelada
		// del movimiento, no una tasa "spot" nueva.
		if models.MeetsRefundMin(refundableBs, rate) && models.IsRefundableCreditMovement(t.MovementType) {
			until := t.CreatedAt.Add(models.RefundRequestWindow)
			untilStr := until.Format(time.RFC3339)
			items[i].RefundEligibleUntil = &untilStr
			if withinRefundWindow(t.CreatedAt, now) && balanceBs >= refundableBs {
				active, err := f.repo.HasActiveRefundForTransaction(ctx, t.ID)
				if err == nil && !active {
					items[i].CanRequestRefund = true
				}
			}
		}
	}

	return dto.CustomerBalanceResponse{
		BalanceBs:    balanceBs,
		BalanceUSD:   balanceUSD,
		BcvRate:      spotRate,
		Transactions: items,
	}, nil
}

func refundableAmount(t models.Transaction) float64 {
	if t.RefundablePMBs != nil {
		return *t.RefundablePMBs
	}
	if models.IsRefundableCreditMovement(t.MovementType) &&
		models.MovementType(t.MovementType) != models.MovementTypeOrderRejectedRefund {
		if t.Amount > 0 {
			return t.Amount
		}
	}
	return 0
}

func (f *customerFinanceApp) GetBalanceBs(ctx context.Context, customerID uuid.UUID) (float64, error) {
	b, err := f.repo.GetBalance(ctx, customerID, models.EntityTypeCustomer)
	if err != nil {
		return 0, err
	}
	return b.Amount, nil
}

func (f *customerFinanceApp) GetBalancePaymentDebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string) (float64, error) {
	return f.repo.SumBalancePaymentDebit(ctx, customerID, referenceID, referenceType)
}

// creditWithRefundable acredita amountBs al ledger (moneda nativa) y congela
// bcvRate en el movimiento para el umbral de devolución y la proyección USD.
func (f *customerFinanceApp) creditWithRefundable(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, refundablePMBs, bcvRate float64, movementType models.MovementType) (uuid.UUID, time.Time, error) {
	if amountBs <= 0 || bcvRate <= 0 {
		return uuid.Nil, time.Time{}, nil
	}
	amountBs = money.RoundBs(amountBs)
	refID := referenceID
	refType := referenceType
	createdAt := time.Now()
	rate := bcvRate
	tx := models.Transaction{
		ID:            uuid.New(),
		ReferenceID:   &refID,
		ReferenceType: &refType,
		MovementType:  string(movementType),
		Currency:      models.CurrencyVES,
		BcvRate:       &rate,
		CreatedAt:     createdAt,
	}
	if refundablePMBs > 0 {
		r := money.RoundBs(refundablePMBs)
		tx.RefundablePMBs = &r
	}
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountBs, 0, tx)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return tx.ID, createdAt, nil
}

func (f *customerFinanceApp) CreditPaymentOverage(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (uuid.UUID, time.Time, error) {
	return f.creditWithRefundable(ctx, customerID, referenceID, referenceType, amountBs, amountBs, bcvRate, models.MovementTypePaymentOverage)
}

func (f *customerFinanceApp) CreditPaymentPartial(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, bcvRate float64) (uuid.UUID, time.Time, error) {
	return f.creditWithRefundable(ctx, customerID, referenceID, referenceType, amountBs, amountBs, bcvRate, models.MovementTypePaymentPartialCredit)
}

func (f *customerFinanceApp) CreditOrderRejected(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs, refundablePMBs, bcvRate float64) (uuid.UUID, time.Time, error) {
	return f.creditWithRefundable(ctx, customerID, referenceID, referenceType, amountBs, refundablePMBs, bcvRate, models.MovementTypeOrderRejectedRefund)
}

// DebitForReference/RefundBalancePayment operan directo en Bs: amountBs es lo que
// el customer pidió usar de su balance (o lo que se le devuelve), ya en su
// moneda nativa — no requiere tasa de conversión.
func (f *customerFinanceApp) DebitForReference(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error {
	if amountBs <= 0 {
		return nil
	}
	refID := referenceID
	refType := referenceType
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, -amountBs, amountBs, models.Transaction{
		ID:            uuid.New(),
		ReferenceID:   &refID,
		ReferenceType: &refType,
		MovementType:  string(models.MovementTypeBalancePayment),
		Currency:      models.CurrencyVES,
	})
	return err
}

func (f *customerFinanceApp) RefundBalancePayment(ctx context.Context, customerID, referenceID uuid.UUID, referenceType string, amountBs float64) error {
	if amountBs <= 0 {
		return nil
	}
	refID := referenceID
	refType := referenceType
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountBs, 0, models.Transaction{
		ID:            uuid.New(),
		ReferenceID:   &refID,
		ReferenceType: &refType,
		MovementType:  string(models.MovementTypeBalancePaymentRefund),
		Currency:      models.CurrencyVES,
	})
	return err
}

func (f *customerFinanceApp) GetPagoMovil(ctx context.Context, customerID uuid.UUID) (dto.PagoMovilResponse, error) {
	pm, err := f.repo.GetPagoMovil(ctx, customerID)
	if err != nil {
		return dto.PagoMovilResponse{}, err
	}
	return toPagoMovilDTO(pm), nil
}

func (f *customerFinanceApp) UpsertPagoMovil(ctx context.Context, customerID uuid.UUID, req dto.UpsertPagoMovilRequest) (dto.PagoMovilResponse, error) {
	phone := strings.TrimSpace(req.Phone)
	bank := strings.TrimSpace(req.BankCode)
	nid := strings.TrimSpace(req.NationalID)
	if phone == "" || bank == "" || nid == "" {
		return dto.PagoMovilResponse{}, errcustom.ErrInvalidBalanceAmount
	}
	pm := &models.CustomerPagoMovil{
		CustomerID: customerID,
		Phone:      phone,
		BankCode:   bank,
		NationalID: nid,
	}
	if err := f.repo.UpsertPagoMovil(ctx, pm); err != nil {
		return dto.PagoMovilResponse{}, err
	}
	saved, err := f.repo.GetPagoMovil(ctx, customerID)
	if err != nil {
		return dto.PagoMovilResponse{}, err
	}
	return toPagoMovilDTO(saved), nil
}

func (f *customerFinanceApp) CreateRefundRequest(ctx context.Context, customerID, transactionID uuid.UUID) (dto.RefundRequestResponse, error) {
	pm, err := f.repo.GetPagoMovil(ctx, customerID)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return dto.RefundRequestResponse{}, errcustom.ErrPagoMovilRequired
		}
		return dto.RefundRequestResponse{}, err
	}

	tx, err := f.repo.GetTransaction(ctx, transactionID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if tx.EntityID != customerID || tx.EntityType != models.EntityTypeCustomer {
		return dto.RefundRequestResponse{}, errcustom.ErrNotFound
	}
	if !models.IsRefundableCreditMovement(tx.MovementType) {
		return dto.RefundRequestResponse{}, errcustom.ErrRefundNotEligible
	}

	amountBs := refundableAmount(*tx)
	rate := 0.0
	if tx.BcvRate != nil {
		rate = *tx.BcvRate
	}
	if !models.MeetsRefundMin(amountBs, rate) {
		return dto.RefundRequestResponse{}, errcustom.ErrRefundNotEligible
	}
	if !withinRefundWindow(tx.CreatedAt, time.Now()) {
		return dto.RefundRequestResponse{}, errcustom.ErrRefundNotEligible
	}

	active, err := f.repo.HasActiveRefundForTransaction(ctx, transactionID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if active {
		return dto.RefundRequestResponse{}, errcustom.ErrRefundAlreadyRequested
	}

	req := &models.BalanceRefundRequest{
		ID:            uuid.New(),
		CustomerID:    customerID,
		TransactionID: transactionID,
		ReferenceID:   tx.ReferenceID,
		ReferenceType: tx.ReferenceType,
		AmountBs:      amountBs,
		Status:        models.RefundPending,
		PMPhone:       pm.Phone,
		PMBankCode:    pm.BankCode,
		PMNationalID:  pm.NationalID,
		CreatedAt:     time.Now(),
		ExpiresAt:     tx.CreatedAt.Add(models.RefundRequestWindow),
	}
	now := time.Now()
	req.LastAlertedAt = &now
	if err := f.repo.CreateRefundRequestWithDebit(ctx, req); err != nil {
		return dto.RefundRequestResponse{}, err
	}

	f.publishRefundPendingAlert(ctx, *req, 0)
	f.broadcastRefund(ctx, *req)
	return toRefundDTO(*req), nil
}

func (f *customerFinanceApp) ListRefundRequests(ctx context.Context, customerID uuid.UUID) ([]dto.RefundRequestResponse, error) {
	list, err := f.repo.ListRefundRequestsByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.RefundRequestResponse, len(list))
	for i := range list {
		out[i] = toRefundDTO(list[i])
	}
	return out, nil
}

func (f *customerFinanceApp) GetRefundRequest(ctx context.Context, customerID, requestID uuid.UUID) (dto.RefundRequestResponse, error) {
	req, err := f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if req.CustomerID != customerID {
		return dto.RefundRequestResponse{}, errcustom.ErrNotFound
	}
	return toRefundDTO(*req), nil
}

func (f *customerFinanceApp) ListActiveForStaff(ctx context.Context) ([]dto.RefundRequestBroadcast, error) {
	list, err := f.repo.ListActiveRefundRequests(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.RefundRequestBroadcast, 0, len(list))
	for i := range list {
		out = append(out, f.toBroadcast(ctx, list[i]))
	}
	return out, nil
}

func (f *customerFinanceApp) AcceptRefund(ctx context.Context, requestID, staffID uuid.UUID) (dto.RefundRequestResponse, error) {
	req, err := f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if req.Status != models.RefundPending {
		return dto.RefundRequestResponse{}, errcustom.ErrInvalidStatus
	}

	ok, err := f.repo.AcceptPending(ctx, requestID, staffID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if !ok {
		return dto.RefundRequestResponse{}, errcustom.ErrInvalidStatus
	}

	req, err = f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	f.broadcastRefund(ctx, *req)
	return toRefundDTO(*req), nil
}

func (f *customerFinanceApp) RejectRefund(ctx context.Context, requestID, staffID uuid.UUID, reason string) (dto.RefundRequestResponse, error) {
	ok, err := f.repo.RejectPendingWithRestore(ctx, requestID, staffID, strings.TrimSpace(reason))
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if !ok {
		return dto.RefundRequestResponse{}, errcustom.ErrInvalidStatus
	}
	req, err := f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	f.broadcastRefund(ctx, *req)
	return toRefundDTO(*req), nil
}

func (f *customerFinanceApp) CompleteRefund(ctx context.Context, requestID uuid.UUID, proofImage string, amountBs float64) (dto.RefundRequestResponse, error) {
	req, err := f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if req.Status != models.RefundAccepted {
		return dto.RefundRequestResponse{}, errcustom.ErrInvalidStatus
	}
	if math.Abs(amountBs-req.AmountBs) > refundAmountTolerance {
		return dto.RefundRequestResponse{}, errcustom.ErrRefundAmountMismatch
	}
	if strings.TrimSpace(proofImage) == "" {
		return dto.RefundRequestResponse{}, errcustom.ErrRefundProofRequired
	}
	if f.uploader == nil {
		return dto.RefundRequestResponse{}, errcustom.ErrUploadingImage
	}
	upload, err := f.uploader.UploadImage(ctx, proofImage)
	if err != nil {
		return dto.RefundRequestResponse{}, errcustom.ErrUploadingImage
	}
	proofURL := upload.SecureURL
	if err := validateRefundProofURL(proofURL); err != nil {
		return dto.RefundRequestResponse{}, err
	}

	ok, err := f.repo.CompleteAccepted(ctx, requestID, proofURL)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if !ok {
		return dto.RefundRequestResponse{}, errcustom.ErrInvalidStatus
	}
	req, err = f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	f.broadcastRefund(ctx, *req)
	return toRefundDTO(*req), nil
}

func (f *customerFinanceApp) broadcastRefund(ctx context.Context, req models.BalanceRefundRequest) {
	payload := f.toBroadcast(ctx, req)
	if f.hub != nil {
		_ = f.hub.Broadcast(payload)
	}
	if f.notifier == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("broadcastRefund marshal failed", "requestID", req.ID, "error", err)
		return
	}
	if err := f.notifier.NotifyCustomer(req.CustomerID, data); err != nil {
		slog.Error("broadcastRefund notify customer failed", "requestID", req.ID, "error", err)
	}
}

func (f *customerFinanceApp) NotifyIncompleteRefunds(ctx context.Context) error {
	if f.stream == nil {
		return nil
	}
	now := time.Now()
	alertBefore := now.Add(-models.RefundAlertInterval)
	list, err := f.repo.ListIncompleteForAlert(ctx, alertBefore)
	if err != nil {
		return err
	}
	for i := range list {
		req := list[i]
		ok, err := f.repo.MarkAlerted(ctx, req.ID, now, alertBefore)
		if err != nil {
			slog.Error("NotifyIncompleteRefunds MarkAlerted failed", "requestID", req.ID, "error", err)
			continue
		}
		if !ok {
			continue
		}
		minutes := max(int(now.Sub(req.CreatedAt)/time.Minute), 0)
		f.publishRefundPendingAlert(ctx, req, minutes)
	}
	return nil
}

func (f *customerFinanceApp) publishRefundPendingAlert(ctx context.Context, req models.BalanceRefundRequest, minutesPending int) {
	if f.stream == nil {
		return
	}
	event := models.DataEventBalanceRefundPending{
		MessageType:    "balance_refund_pending",
		RequestID:      req.ID.String(),
		MinutesPending: minutesPending,
	}
	if err := f.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("publishRefundPendingAlert failed", "requestID", req.ID, "error", err)
		return
	}
	slog.Info("balance_refund_pending published",
		"requestID", req.ID,
		"minutesPending", minutesPending,
		"status", req.Status,
	)
}

func orderIDFromReference(id *uuid.UUID, referenceType *string) *string {
	return idFromReference(id, referenceType, paymentsModels.ReferenceTypeOrder)
}

func tripIDFromReference(id *uuid.UUID, referenceType *string) *string {
	return idFromReference(id, referenceType, paymentsModels.ReferenceTypeTrip)
}

func idFromReference(id *uuid.UUID, referenceType *string, want string) *string {
	if id == nil || referenceType == nil || *referenceType != want {
		return nil
	}
	s := id.String()
	return &s
}

func toPagoMovilDTO(pm *models.CustomerPagoMovil) dto.PagoMovilResponse {
	return dto.PagoMovilResponse{
		Phone:      pm.Phone,
		BankCode:   pm.BankCode,
		NationalID: pm.NationalID,
		UpdatedAt:  pm.UpdatedAt.Format(time.RFC3339),
	}
}

func toRefundDTO(req models.BalanceRefundRequest) dto.RefundRequestResponse {
	out := dto.RefundRequestResponse{
		ID:              req.ID.String(),
		TransactionID:   req.TransactionID.String(),
		AmountBs:        req.AmountBs,
		Status:          string(req.Status),
		PMPhone:         req.PMPhone,
		PMBankCode:      req.PMBankCode,
		PMNationalID:    req.PMNationalID,
		ProofURL:        req.ProofURL,
		RejectionReason: req.RejectionReason,
		CreatedAt:       req.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       req.ExpiresAt.Format(time.RFC3339),
	}
	if orderID := orderIDFromReference(req.ReferenceID, req.ReferenceType); orderID != nil {
		out.OrderID = orderID
	}
	if tripID := tripIDFromReference(req.ReferenceID, req.ReferenceType); tripID != nil {
		out.TripID = tripID
	}
	if req.AcceptedAt != nil {
		s := req.AcceptedAt.Format(time.RFC3339)
		out.AcceptedAt = &s
	}
	if req.CompletedAt != nil {
		s := req.CompletedAt.Format(time.RFC3339)
		out.CompletedAt = &s
	}
	if req.ResolvedAt != nil {
		s := req.ResolvedAt.Format(time.RFC3339)
		out.ResolvedAt = &s
	}
	return out
}

func (f *customerFinanceApp) toBroadcast(ctx context.Context, req models.BalanceRefundRequest) dto.RefundRequestBroadcast {
	b := dto.RefundRequestBroadcast{
		Type:            "refund_request_update",
		ID:              req.ID.String(),
		CustomerID:      req.CustomerID.String(),
		TransactionID:   req.TransactionID.String(),
		AmountBs:        req.AmountBs,
		Status:          string(req.Status),
		PMPhone:         req.PMPhone,
		PMBankCode:      req.PMBankCode,
		PMNationalID:    req.PMNationalID,
		ProofURL:        req.ProofURL,
		RejectionReason: req.RejectionReason,
		CreatedAt:       req.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       req.ExpiresAt.Format(time.RFC3339),
	}
	if orderID := orderIDFromReference(req.ReferenceID, req.ReferenceType); orderID != nil {
		b.OrderID = orderID
	}
	if tripID := tripIDFromReference(req.ReferenceID, req.ReferenceType); tripID != nil {
		b.TripID = tripID
	}
	f.applyAmountUSD(ctx, &b, req.AmountBs)
	return b
}

// applyAmountUSD agrega el equivalente informativo en USD (tasa BCV vigente
// al momento de armar el broadcast, no la tasa original del movimiento) para
// que el panel de staff pueda conciliar contra el total de la orden en USD.
func (f *customerFinanceApp) spotBcvRate(ctx context.Context) float64 {
	if f.bcv == nil {
		return 0
	}
	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		if err != nil {
			slog.Warn("balance dual currency: bcv rate unavailable", "error", err)
		}
		return 0
	}
	return rate.Rate
}

func (f *customerFinanceApp) applyAmountUSD(ctx context.Context, b *dto.RefundRequestBroadcast, amountBs float64) {
	if f.bcv == nil || amountBs <= 0 {
		return
	}
	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		if err != nil {
			slog.Warn("refund amount_usd: bcv rate unavailable", "error", err)
		}
		return
	}
	b.BcvRate = rate.Rate
	b.AmountUSD = money.RoundUSD(amountBs / rate.Rate)
}
