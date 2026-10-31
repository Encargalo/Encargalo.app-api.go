package code

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateFourDigits genera un código de verificación de 4 dígitos (0000-9999),
// usado por orders, trips y shipments para confirmar retiro/entrega en persona.
func GenerateFourDigits() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}
