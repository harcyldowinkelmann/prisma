package models

import "github.com/google/uuid"

type Transaction struct {
	UUID          uuid.UUID `json:"id"`
	Description   string    `json:"description"`
	Amount        float64   `json:"amount"`
	Date          string    `json:"date"`
	Category      string    `json:"category"`
	Subcategory   string    `json:"subcategory"`
	PaymentMethod string    `json:"payment_method"`
	Installments  string    `json:"installments"`
	Tags          string    `json:"tags"`
	IsPaid        bool      `json:"is_paid"`
	Active        bool      `json:"active"`
}

type TransactionFilters struct {
	Description *string  `json:"description"`
	Amount      *float64 `json:"amount"`
	Date        *string  `json:"date"`
	Category    *string  `json:"category"`
}

// SettingItem is a generic model for settings like subcategories, payment methods, and tags.
type SettingItem struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
