package models

type AvailabilityStatus string

const (
	AvailabilityActive     AvailabilityStatus = "active"
	AvailabilityHidden     AvailabilityStatus = "hidden"
	AvailabilityOutOfStock AvailabilityStatus = "out_of_stock"
)

func IsValidAvailabilityStatus(value string) bool {
	switch AvailabilityStatus(value) {
	case AvailabilityActive, AvailabilityHidden, AvailabilityOutOfStock:
		return true
	default:
		return false
	}
}

func (s AvailabilityStatus) IsVisibleToCustomer() bool {
	return s != AvailabilityHidden
}

func (s AvailabilityStatus) CanOrder() bool {
	return s == AvailabilityActive
}

func NormalizeAvailabilityStatus(value string) AvailabilityStatus {
	if IsValidAvailabilityStatus(value) {
		return AvailabilityStatus(value)
	}
	return AvailabilityActive
}
