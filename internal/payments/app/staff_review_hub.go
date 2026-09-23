package app

import (
	"context"
	"encoding/json"
	"sync"

	"Encargalo.app-api.go/internal/payments/domain/models"
	"Encargalo.app-api.go/internal/payments/domain/ports"
	"github.com/google/uuid"
)

// StaffReviewHubImpl is the concrete hub. dig needs the concrete type so main
// can call SetSnapshotProvider after the container is built.
type StaffReviewHubImpl struct {
	mu          sync.RWMutex
	connections map[uuid.UUID][]ports.ManagerConn
	providers   []func(ctx context.Context) ([]models.PaymentReviewBroadcast, error)
	repo        ports.PaymentReviewsRepo
}

func NewStaffReviewHub(repo ports.PaymentReviewsRepo) *StaffReviewHubImpl {
	return &StaffReviewHubImpl{
		connections: make(map[uuid.UUID][]ports.ManagerConn),
		repo:        repo,
	}
}

func (h *StaffReviewHubImpl) SetSnapshotProvider(fn func(ctx context.Context) ([]models.PaymentReviewBroadcast, error)) {
	if fn == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.providers = append(h.providers, fn)
}

func (h *StaffReviewHubImpl) Register(ctx context.Context, staffID uuid.UUID, conn ports.ManagerConn) error {
	h.mu.Lock()
	h.connections[staffID] = append(h.connections[staffID], conn)
	providers := append([]func(context.Context) ([]models.PaymentReviewBroadcast, error){}, h.providers...)
	h.mu.Unlock()

	items, err := h.snapshot(ctx, providers)
	if err != nil {
		return err
	}
	for _, item := range items {
		data, _ := json.Marshal(item)
		conn.Send(data)
	}
	return nil
}

func (h *StaffReviewHubImpl) snapshot(ctx context.Context, providers []func(context.Context) ([]models.PaymentReviewBroadcast, error)) ([]models.PaymentReviewBroadcast, error) {
	if len(providers) == 0 {
		return h.genericSnapshot(ctx)
	}
	var out []models.PaymentReviewBroadcast
	for _, fn := range providers {
		items, err := fn(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func (h *StaffReviewHubImpl) genericSnapshot(ctx context.Context) ([]models.PaymentReviewBroadcast, error) {
	if h.repo == nil {
		return nil, nil
	}
	reviews, err := h.repo.ListActiveForStaff(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.PaymentReviewBroadcast, 0, len(reviews))
	for i := range reviews {
		out = append(out, genericBroadcast(&reviews[i]))
	}
	return out, nil
}

func (h *StaffReviewHubImpl) Unregister(staffID uuid.UUID, conn ports.ManagerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	list, ok := h.connections[staffID]
	if !ok {
		conn.Close()
		return
	}

	filtered := list[:0]
	for _, c := range list {
		if c != conn {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		delete(h.connections, staffID)
	} else {
		h.connections[staffID] = filtered
	}
	conn.Close()
}

func (h *StaffReviewHubImpl) BroadcastReview(review models.PaymentReviewBroadcast) error {
	data, err := json.Marshal(review)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.connections {
		for _, c := range clients {
			c.Send(data)
		}
	}
	return nil
}

func genericBroadcast(review *models.PaymentReview) models.PaymentReviewBroadcast {
	return models.PaymentReviewBroadcast{
		Type:             "payment_review_update",
		ReviewID:         review.ID,
		ReferenceID:      review.ReferenceID,
		ReferenceType:    review.ReferenceType,
		ProofURL:         review.ProofURL,
		ExpectedAmountBs: review.ExpectedAmountBs,
		ObservedAmountBs: review.ObservedAmountBs,
		DifferenceBs:     review.DifferenceBs,
		Status:           review.Status,
		RejectionReason:  review.RejectionReason,
		CreditedBs:       review.CreditedBs,
		CreatedAt:        review.CreatedAt,
	}
}

var _ ports.StaffReviewHub = (*StaffReviewHubImpl)(nil)
