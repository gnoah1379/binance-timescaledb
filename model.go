package main

import "time"

type Candlestick struct {
	Time   time.Time `json:"time" db:"time"`
	Symbol string    `json:"symbol" db:"symbol"`
	Open   float64   `json:"open" db:"open"`
	High   float64   `json:"high" db:"high"`
	Low    float64   `json:"low" db:"low"`
	Close  float64   `json:"close" db:"close"`
	Volume float64   `json:"volume" db:"volume"`
}

type Symbol struct {
	Symbol  string `json:"symbol" db:"symbol"`
	Name    string `json:"name" db:"name"`
	Enabled bool   `json:"enabled" db:"enabled"`
}
