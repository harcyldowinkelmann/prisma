package models

import "github.com/google/uuid"

type Transaction struct {
	UUID        uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Date        string    `json:"date"`
	Category    string    `json:"category"`
	Active      bool      `json:"active"`
}

type TransactionFilters struct {
	Description *string  `json:"description"`
	Amount      *float64 `json:"amount"`
	Date        *string  `json:"date"`
	Category    *string  `json:"category"`
}
