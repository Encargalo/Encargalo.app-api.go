package app

import (
	"context"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/customers/domain/dto"
	portsCusto "Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/pkg/logs"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
)

const (
	typeCustomer = "customer"
)

type appAuth struct {
	svc       portsCusto.CustomersApp
	bycript   bycript.Password
	repo      ports.AuthRepo
	slackLogs logs.Logs
}

func NewAuthApp(svc portsCusto.CustomersApp, bycript bycript.Password, repo ports.AuthRepo, slackLogs logs.Logs) ports.AuthApp {
	return &appAuth{svc, bycript, repo, slackLogs}
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

func (a *appAuth) SearchSessions(ctx context.Context, session_id uuid.UUID) (*models.ActiveSession, error) {
	return a.repo.SearchSession(ctx, session_id)
}

func (a *appAuth) DeleteSession(ctx context.Context, session_id uuid.UUID) error {

	session, err := a.SearchSessions(ctx, session_id)
	if err != nil {
		return err
	}

	if session == nil {
		slog.Warn("session " + session_id.String() + "not found from delete")
		return nil
	}

	if err := a.repo.DeleteSession(ctx, session_id); err != nil {
		return err
	}

	return nil
}
