package app

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func generateTripCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}
