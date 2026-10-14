package app

import (
	"context"
	"errors"
	"slices"

	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/adapters/googleauth"
	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	portsCusto "Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

type appAuth struct {
	svc            portsCusto.UsersApp
	bycript        bycript.Password
	repo           ports.AuthRepo
	googleVerifier googleauth.Verifier
	riderProfiles  ridersPorts.RiderProfileApp
}

func NewAuthApp(
	svc portsCusto.UsersApp,
	bycript bycript.Password,
	repo ports.AuthRepo,
	googleVerifier googleauth.Verifier,
	riderProfiles ridersPorts.RiderProfileApp,
) ports.AuthApp {
	return &appAuth{svc, bycript, repo, googleVerifier, riderProfiles}
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

	if err := a.ensureAccountEnabled(ctx, custo.ID); err != nil {
		return uuid.Nil, err
	}

	a.cancelPendingDeletion(ctx, custo.ID)

	ip, userAgent := requestinfo.FromContext(ctx)

	var session models.ActiveSession
	session.BuildActiveSessionModel(custo.ID, string(models.Customer), []models.Role{models.Customer}, ip, userAgent, nil)

	if err := a.repo.SaveSession(ctx, &session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}

// cancelPendingDeletion revierte, en modo best-effort, una solicitud de
// eliminación de cuenta pendiente: si el usuario vuelve a iniciar sesión
// dentro del período de gracia, la solicitud se cancela sola. Un fallo
// transitorio acá no debe bloquear el login — se reintentará en el próximo.
func (a *appAuth) cancelPendingDeletion(ctx context.Context, userID uuid.UUID) {
	if err := a.svc.CancelPendingDeletionIfAny(ctx, userID); err != nil {
		applog.Warn(ctx, "no se pudo cancelar la eliminación pendiente de la cuenta", err, "user_id", userID.String())
	}
}

func (a *appAuth) SignInEmployee(ctx context.Context, phone, password string) (uuid.UUID, error) {

	empl, err := a.svc.SearchAccountByPhone(ctx, phone, []string{"Order Manager", "Owner"})
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return uuid.Nil, errcustom.ErrIncorrectAccessData
		}
		return uuid.Nil, errcustom.ErrUnexpectedError
	}

	if !a.bycript.CheckPasswordHash([]byte(empl.Password), password) {
		return uuid.Nil, errcustom.ErrIncorrectAccessData
	}

	if err := a.ensureAccountEnabled(ctx, empl.ID); err != nil {
		return uuid.Nil, err
	}

	a.cancelPendingDeletion(ctx, empl.ID)

	ip, userAgent := requestinfo.FromContext(ctx)

	var session models.ActiveSession
	session.BuildActiveSessionModel(empl.ID, empl.Role, []models.Role{models.Role(empl.Role)}, ip, userAgent, &empl.ShopID)

	if err := a.repo.SaveSession(ctx, &session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}

func (a *appAuth) SignInRider(ctx context.Context, phone, password string) (uuid.UUID, error) {
	rider, err := a.svc.SearchAccountByPhone(ctx, phone, []string{string(models.Rider)})
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return uuid.Nil, errcustom.ErrIncorrectAccessData
		}
		return uuid.Nil, errcustom.ErrUnexpectedError
	}

	if !a.bycript.CheckPasswordHash([]byte(rider.Password), password) {
		return uuid.Nil, errcustom.ErrIncorrectAccessData
	}

	if err := a.ensureAccountEnabled(ctx, rider.ID); err != nil {
		return uuid.Nil, err
	}

	if a.riderProfiles != nil {
		active, err := a.riderProfiles.IsActiveRider(ctx, rider.ID)
		if err != nil {
			return uuid.Nil, errcustom.ErrUnexpectedError
		}
		if !active {
			return uuid.Nil, errcustom.ErrRiderNotActive
		}
	}

	a.cancelPendingDeletion(ctx, rider.ID)

	ip, userAgent := requestinfo.FromContext(ctx)

	var session models.ActiveSession
	session.BuildActiveSessionModel(rider.ID, string(models.Rider), []models.Role{models.Rider}, ip, userAgent, nil)

	if err := a.repo.SaveSession(ctx, &session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}

// IdentifyRiderPhone dice qué pantalla sigue en la app de riders.
// No abre sesión ni revela datos de la cuenta.
func (a *appAuth) IdentifyRiderPhone(ctx context.Context, phone string) (models.RiderPhoneNext, error) {
	account, err := a.svc.SearchCustomerBy(ctx, dto.SearchCustomerBy{Phone: phone})
	if errors.Is(err, errcustom.ErrNotFound) {
		return models.RiderPhoneNextRegister, nil
	}
	if err != nil {
		return "", errcustom.ErrUnexpectedError
	}

	roles, err := a.svc.GetUserRoles(ctx, account.ID)
	if err != nil {
		return "", errcustom.ErrUnexpectedError
	}

	if slices.Contains(roles, string(models.Rider)) {
		return models.RiderPhoneNextSignIn, nil
	}

	return models.RiderPhoneNextOnboarding, nil
}

func (a *appAuth) SignInStaff(ctx context.Context, phone, password string) (uuid.UUID, error) {
	staff, err := a.svc.SearchAccountByPhone(ctx, phone, []string{string(models.Staff)})
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return uuid.Nil, errcustom.ErrIncorrectAccessData
		}
		return uuid.Nil, errcustom.ErrUnexpectedError
	}

	if !a.bycript.CheckPasswordHash([]byte(staff.Password), password) {
		return uuid.Nil, errcustom.ErrIncorrectAccessData
	}

	if err := a.ensureAccountEnabled(ctx, staff.ID); err != nil {
		return uuid.Nil, err
	}

	a.cancelPendingDeletion(ctx, staff.ID)

	ip, userAgent := requestinfo.FromContext(ctx)

	var session models.ActiveSession
	session.BuildActiveSessionModel(staff.ID, string(models.Staff), []models.Role{models.Staff}, ip, userAgent, nil)

	if err := a.repo.SaveSession(ctx, &session); err != nil {
		return uuid.Nil, err
	}

	return session.ID, nil
}

func (a *appAuth) SignInGoogle(ctx context.Context, idToken string) (uuid.UUID, error) {

	userInfo, err := a.googleVerifier.Verify(ctx, idToken)
	if err != nil {
		return uuid.Nil, errcustom.ErrUnautorized
	}

	custo, err := a.svc.SearchCustomerBy(ctx, dto.SearchCustomerBy{Email: userInfo.Email})
	if err != nil {
		if !errors.Is(err, errcustom.ErrNotFound) {
			return uuid.Nil, errcustom.ErrUnexpectedError
		}

		custo, err = a.svc.RegisterCustomerFromGoogle(ctx, dto.RegisterCustomerFromGoogle{
			Name:    userInfo.Name,
			SurName: userInfo.SurName,
			Email:   userInfo.Email,
		})
		if err != nil {
			return uuid.Nil, err
		}
	}

	if err := a.ensureAccountEnabled(ctx, custo.ID); err != nil {
		return uuid.Nil, err
	}

	a.cancelPendingDeletion(ctx, custo.ID)

	ip, userAgent := requestinfo.FromContext(ctx)

	var session models.ActiveSession
	session.BuildActiveSessionModel(custo.ID, string(models.Customer), []models.Role{models.Customer}, ip, userAgent, nil)

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
		applog.Warn(ctx, "sesion no encontrada al intentar eliminarla", nil, "session_id", session_id.String())
		return nil
	}

	if err := a.repo.DeleteSession(ctx, session_id); err != nil {
		return err
	}

	return nil
}

func (a *appAuth) DeleteAllSessionsForUser(ctx context.Context, userID uuid.UUID) error {
	return a.repo.DeleteAllSessionsForUser(ctx, userID)
}

func (a *appAuth) EnsureAccountEnabled(ctx context.Context, userID uuid.UUID) error {
	return a.ensureAccountEnabled(ctx, userID)
}

func (a *appAuth) ensureAccountEnabled(ctx context.Context, userID uuid.UUID) error {
	acc, err := a.svc.SearchCustomerBy(ctx, dto.SearchCustomerBy{ID: userID})
	if err != nil {
		if errors.Is(err, errcustom.ErrNotFound) {
			return errcustom.ErrIncorrectAccessData
		}
		return errcustom.ErrUnexpectedError
	}
	if !acc.Enabled {
		return errcustom.ErrAccountDisabled
	}
	return nil
}
