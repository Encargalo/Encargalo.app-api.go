package models

import "time"

type ExchangeRate struct {
	Rate      float64   `json:"rate"`
	Date      time.Time `json:"date"`
	FetchedAt time.Time `json:"fetched_at"`
}
