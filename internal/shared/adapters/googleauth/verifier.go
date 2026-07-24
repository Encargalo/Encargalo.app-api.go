package googleauth

import (
	"context"
	"errors"
	"fmt"

	"Encargalo.app-api.go/internal/shared/config"
	"google.golang.org/api/idtoken"
)

// UserInfo son los datos de identidad que Google confirma al validar el id_token.
type UserInfo struct {
	Email   string
	Name    string
	SurName string
}

type Verifier interface {
	Verify(ctx context.Context, idToken string) (*UserInfo, error)
}

type verifier struct{}

func NewVerifier() Verifier {
	return &verifier{}
}

func (v *verifier) Verify(ctx context.Context, idToken string) (*UserInfo, error) {
	payload, err := idtoken.Validate(ctx, idToken, config.Get().Google.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid google id_token: %w", err)
	}

	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		return nil, errors.New("google id_token email is not verified")
	}

	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return nil, errors.New("google id_token has no email claim")
	}

	name, _ := payload.Claims["given_name"].(string)
	surName, _ := payload.Claims["family_name"].(string)

	return &UserInfo{Email: email, Name: name, SurName: surName}, nil
}
