package money

import "math"

// RoundBs redondea un monto en bolívares a 2 decimales (ej. 500.21).
// Se aplica sobre el resultado de multiplicar un precio en USD por la tasa BCV.
func RoundBs(amount float64) float64 {
	return math.Round(amount*100) / 100
}
