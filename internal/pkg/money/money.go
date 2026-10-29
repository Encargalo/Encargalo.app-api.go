package money

import "math"

// RoundBs redondea un monto en bolívares a la decena más cercana (ej. 550).
// Se aplica sobre el resultado de multiplicar un precio en USD por la tasa BCV.
func RoundBs(amount float64) float64 {
	return math.Round(amount/10) * 10
}

// RoundUSD redondea un monto en dólares a 2 decimales.
func RoundUSD(amount float64) float64 {
	return math.Round(amount*100) / 100
}
