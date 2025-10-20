package jwt

import (
	"fmt"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claim struct {
	SessionID uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}

type Sessions interface {
	CreateSession(sessionID uuid.UUID) (string, error)
	ValidateToken(value string) (jwt.MapClaims, error)
}

type sessions struct {
	config config.Config
}

func NewSessionUtils(config config.Config) Sessions {
	return &sessions{config}
}

func (s *sessions) CreateSession(sessionID uuid.UUID) (string, error) {

	configJWT := s.config.JWT

	claims := Claim{
		SessionID:        sessionID,
		RegisteredClaims: jwt.RegisteredClaims{},
	}

	claims.ExpiresAt = jwt.NewNumericDate(time.Now().AddDate(1, 0, 0))

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(configJWT.Secret))

}

func (s *sessions) ValidateToken(value string) (jwt.MapClaims, error) {

	token, err := jwt.Parse(value, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.config.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
