package app

import (
	"sync"

	"Encargalo.app-api.go/internal/payments/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
)

// SubjectRegistry is exported so cmd/server can register subjects after dig builds the container.
type SubjectRegistry struct {
	mu       sync.RWMutex
	subjects map[string]ports.PaymentSubject
}

func NewSubjectRegistry() *SubjectRegistry {
	return &SubjectRegistry{subjects: make(map[string]ports.PaymentSubject)}
}

func (r *SubjectRegistry) Register(referenceType string, subject ports.PaymentSubject) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subjects[referenceType] = subject
}

func (r *SubjectRegistry) Get(referenceType string) (ports.PaymentSubject, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.subjects[referenceType]
	if !ok {
		return nil, errcustom.ErrInvalidStatus
	}
	return s, nil
}

var _ ports.PaymentSubjectRegistry = (*SubjectRegistry)(nil)
