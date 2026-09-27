package models

import "errors"

const tenderToleranceBs = 0.01

var (
	ErrTooManyTenders     = errors.New("solo se pueden combinar dos métodos de pago")
	ErrTenderExceedsTotal = errors.New("los montos superan el total")
)

// TenderSplit es el desglose en Bs de un pago. Como máximo dos partes son
// mayores que cero.
type TenderSplit struct {
	BalanceBs   float64
	CashBs      float64
	PagoMovilBs float64
}

// ResolveTenders reparte el total. Si el cliente omite una pata, el método
// indica cuál es el resto: Efectivo deja el resto en efectivo, Balance en
// saldo y Pago Móvil en el comprobante.
func ResolveTenders(totalBs, balanceIn, cashIn float64, method string) (TenderSplit, error) {
	if balanceIn < -tenderToleranceBs || cashIn < -tenderToleranceBs {
		return TenderSplit{}, ErrTenderExceedsTotal
	}
	balance, cash := balanceIn, cashIn
	if balance < 0 {
		balance = 0
	}
	if cash < 0 {
		cash = 0
	}

	balance, cash = fillOmittedLeg(totalBs, balance, cash, method)

	pm := totalBs - balance - cash
	if pm < -tenderToleranceBs {
		return TenderSplit{}, ErrTenderExceedsTotal
	}
	if pm < 0 {
		pm = 0
	}

	active := 0
	for _, part := range []float64{balance, cash, pm} {
		if part > tenderToleranceBs {
			active++
		}
	}
	if active == 0 || active > 2 {
		return TenderSplit{}, ErrTooManyTenders
	}
	return TenderSplit{BalanceBs: balance, CashBs: cash, PagoMovilBs: pm}, nil
}

func fillOmittedLeg(totalBs, balance, cash float64, method string) (float64, float64) {
	if method == "Efectivo" {
		return balance, omittedRemainder(totalBs, balance, cash)
	}
	if method == "Balance" {
		return omittedRemainder(totalBs, cash, balance), cash
	}
	return balance, cash
}

// omittedRemainder llena omitted cuando es ~0 y covered no llega al total.
// covered es la pata ya indicada; omitted es la que el método representa.
func omittedRemainder(totalBs, covered, omitted float64) float64 {
	if omitted > tenderToleranceBs {
		return omitted
	}
	if covered <= tenderToleranceBs {
		return totalBs
	}
	if covered >= totalBs-tenderToleranceBs {
		return omitted
	}
	return totalBs - covered
}

// IsPair es true cuando hay exactamente dos partes mayores que cero.
func (s TenderSplit) IsPair() bool {
	active := 0
	for _, part := range []float64{s.BalanceBs, s.CashBs, s.PagoMovilBs} {
		if part > tenderToleranceBs {
			active++
		}
	}
	return active == 2
}
