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
	balance, err := f.repo.GetBalance(ctx, customerID, models.EntityTypeCustomer)
	if err != nil {
		return dto.CustomerBalanceResponse{}, err
	}

	txs, _, err := f.repo.ListTransactions(ctx, customerID, models.EntityTypeCustomer, 1, 10)
	if err != nil {
		return dto.CustomerBalanceResponse{}, err
	}

	now := time.Now()
	items := make([]dto.CustomerTransactionItem, len(txs))
	for i, t := range txs {
		items[i] = dto.CustomerTransactionItem{
			ID:           t.ID.String(),
			Amount:       t.Amount,
			MovementType: t.MovementType,
			CreatedAt:    t.CreatedAt.Format(time.RFC3339),
		}
		if t.OrderID != nil {
			s := t.OrderID.String()
			items[i].OrderID = &s
		}
		if t.RefundRequestID != nil {
			s := t.RefundRequestID.String()
			items[i].RefundRequestID = &s
		}
		if t.RefundablePMUSD != nil {
			v := *t.RefundablePMUSD
			items[i].RefundablePMUSD = &v
		}

		refundable := refundableAmount(t)
		// can_request_refund = elegible por ventana/monto/saldo. El débito de la
		// solicitud ocurre al crear; el balance ya refleja holds pending/accepted.
		if refundable > models.RefundMinAmountUSD && models.IsRefundableCreditMovement(t.MovementType) {
			until := t.CreatedAt.Add(models.RefundRequestWindow)
			untilStr := until.Format(time.RFC3339)
			items[i].RefundEligibleUntil = &untilStr
			if withinRefundWindow(t.CreatedAt, now) && balance >= refundable {
				active, err := f.repo.HasActiveRefundForTransaction(ctx, t.ID)
				if err == nil && !active {
					items[i].CanRequestRefund = true
				}
			}
		}
	}

	return dto.CustomerBalanceResponse{
		Balance:      balance,
		Transactions: items,
	}, nil
}

func refundableAmount(t models.Transaction) float64 {
	if t.RefundablePMUSD != nil {
		return *t.RefundablePMUSD
	}
	if models.IsRefundableCreditMovement(t.MovementType) &&
		models.MovementType(t.MovementType) != models.MovementTypeOrderRejectedRefund {
		if t.Amount > 0 {
			return t.Amount
		}
	}
	return 0
}

func (f *customerFinanceApp) GetBalanceUSD(ctx context.Context, customerID uuid.UUID) (float64, error) {
	return f.repo.GetBalance(ctx, customerID, models.EntityTypeCustomer)
}

func (f *customerFinanceApp) GetBalancePaymentDebitForOrder(ctx context.Context, customerID, orderID uuid.UUID) (float64, error) {
	return f.repo.SumBalancePaymentDebit(ctx, customerID, orderID)
}

func (f *customerFinanceApp) creditWithRefundable(ctx context.Context, customerID, orderID uuid.UUID, amountUSD, refundablePM float64, movementType models.MovementType) (uuid.UUID, time.Time, error) {
	if amountUSD <= 0 {
		return uuid.Nil, time.Time{}, nil
	}
	oid := orderID
	createdAt := time.Now()
	tx := models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(movementType),
		CreatedAt:    createdAt,
	}
	if refundablePM > 0 {
		r := refundablePM
		tx.RefundablePMUSD = &r
	}
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountUSD, 0, tx)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return tx.ID, createdAt, nil
}

func (f *customerFinanceApp) CreditPaymentOverage(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) (uuid.UUID, time.Time, error) {
	return f.creditWithRefundable(ctx, customerID, orderID, amountUSD, amountUSD, models.MovementTypePaymentOverage)
}

func (f *customerFinanceApp) CreditPaymentPartial(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) (uuid.UUID, time.Time, error) {
	return f.creditWithRefundable(ctx, customerID, orderID, amountUSD, amountUSD, models.MovementTypePaymentPartialCredit)
}

func (f *customerFinanceApp) CreditOrderRejected(ctx context.Context, customerID, orderID uuid.UUID, amountUSD, refundablePMUSD float64) (uuid.UUID, time.Time, error) {
	return f.creditWithRefundable(ctx, customerID, orderID, amountUSD, refundablePMUSD, models.MovementTypeOrderRejectedRefund)
}

func (f *customerFinanceApp) DebitForOrder(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, -amountUSD, amountUSD, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypeBalancePayment),
	})
	return err
}

func (f *customerFinanceApp) RefundBalancePayment(ctx context.Context, customerID, orderID uuid.UUID, amountUSD float64) error {
	if amountUSD <= 0 {
		return nil
	}
	oid := orderID
	_, err := f.repo.RecordMovement(ctx, customerID, models.EntityTypeCustomer, amountUSD, 0, models.Transaction{
		ID:           uuid.New(),
		OrderID:      &oid,
		MovementType: string(models.MovementTypeBalancePaymentRefund),
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

	amount := refundableAmount(*tx)
	if amount <= models.RefundMinAmountUSD {
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
		OrderID:       tx.OrderID,
		AmountUSD:     amount,
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

func (f *customerFinanceApp) CompleteRefund(ctx context.Context, requestID uuid.UUID, proofImage string, amountUSD float64) (dto.RefundRequestResponse, error) {
	req, err := f.repo.GetRefundRequest(ctx, requestID)
	if err != nil {
		return dto.RefundRequestResponse{}, err
	}
	if req.Status != models.RefundAccepted {
		return dto.RefundRequestResponse{}, errcustom.ErrInvalidStatus
	}
	if math.Abs(amountUSD-req.AmountUSD) > refundAmountTolerance {
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
		AmountUSD:       req.AmountUSD,
		Status:          string(req.Status),
		PMPhone:         req.PMPhone,
		PMBankCode:      req.PMBankCode,
		PMNationalID:    req.PMNationalID,
		ProofURL:        req.ProofURL,
		RejectionReason: req.RejectionReason,
		CreatedAt:       req.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       req.ExpiresAt.Format(time.RFC3339),
	}
	if req.OrderID != nil {
		s := req.OrderID.String()
		out.OrderID = &s
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
		AmountUSD:       req.AmountUSD,
		Status:          string(req.Status),
		PMPhone:         req.PMPhone,
		PMBankCode:      req.PMBankCode,
		PMNationalID:    req.PMNationalID,
		ProofURL:        req.ProofURL,
		RejectionReason: req.RejectionReason,
		CreatedAt:       req.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       req.ExpiresAt.Format(time.RFC3339),
	}
	if req.OrderID != nil {
		s := req.OrderID.String()
		b.OrderID = &s
	}
	f.applyAmountBs(ctx, &b, req.AmountUSD)
	return b
}

func (f *customerFinanceApp) applyAmountBs(ctx context.Context, b *dto.RefundRequestBroadcast, amountUSD float64) {
	if f.bcv == nil || amountUSD <= 0 {
		return
	}
	rate, err := f.bcv.GetRate(ctx)
	if err != nil || rate == nil || rate.Rate <= 0 {
		if err != nil {
			slog.Warn("refund amount_bs: bcv rate unavailable", "error", err)
		}
		return
	}
	b.BcvRate = rate.Rate
	b.AmountBs = money.RoundBs(amountUSD * rate.Rate)
}
