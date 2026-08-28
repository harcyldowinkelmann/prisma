package models

import "github.com/google/uuid"

// MaxSafeAmountCents is the largest cent value that JavaScript can represent exactly.
const MaxSafeAmountCents int64 = 9007199254740991

// Transaction represents an income or expense stored by Prisma.
type Transaction struct {
	UUID          uuid.UUID `json:"id"`
	Description   string    `json:"description"`
	AmountCents   int64     `json:"amount_cents"`
	Date          string    `json:"date"`
	Category      string    `json:"category"`
	Subcategory   string    `json:"subcategory"`
	PaymentMethod string    `json:"payment_method"`
	Installments  string    `json:"installments"`
	Tags          string    `json:"tags"`
	IsPaid        bool      `json:"is_paid"`
	Reconciled    bool      `json:"reconciled"`
	Active        bool      `json:"active"`
}

// TransactionFilters contains the optional criteria for transaction searches.
type TransactionFilters struct {
	Description     *string `json:"description"`
	AmountCents     *int64  `json:"amount_cents"`
	Date            *string `json:"date"`
	StartDate       *string `json:"start_date"`
	EndDate         *string `json:"end_date"`
	Category        *string `json:"category"`
	IsPaid          *bool   `json:"is_paid"`
	Reconciled      *bool   `json:"reconciled"`
	IncludeArchived bool    `json:"include_archived"`
}

// SettingItem is a generic model for settings like subcategories, payment methods, and tags.
type SettingItem struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
