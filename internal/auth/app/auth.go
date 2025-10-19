package app

import (
	"context"
	"errors"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/customers/domain/dto"
	portsCusto "Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

const (
	typeCustomer = "customer"
)

type appAuth struct {
	svc     portsCusto.CustomersApp
	bycript bycript.Password
	repo    ports.AuthRepo
}

func NewAuthApp(svc portsCusto.CustomersApp, bycript bycript.Password, repo ports.AuthRepo) ports.AuthApp {
	return &appAuth{svc, bycript, repo}
}

func (a *appAuth) SignInCustomer(ctx context.Context, phone, password string) (uuid.UUID, error) {

	criteria := dto.SearchCustomerBy{
		Phone: phone,
	}

	custo, err := a.svc.SearchCustomerBy(ctx, criteria)
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return uuid.Nil, errcustom.ErrIncorrectAccessData
		}

		return uuid.Nil, errcustom.ErrUnexpectedError
	}

	if !a.bycript.CheckPasswordHash([]byte(custo.Password), password) {
		return uuid.Nil, errcustom.ErrIncorrectAccessData
	}

	var session models.ActiveSession
	session.BuildActiveSessionModel(custo.ID, typeCustomer, "", "")

	if err := a.repo.SaveSession(ctx, &session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}
