package models

// RecurringSchedule defines transactions generated on a fixed cadence.
type RecurringSchedule struct {
	UUID          string `json:"uuid"`
	Description   string `json:"description"`
	AmountCents   int64  `json:"amount_cents"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Frequency     string `json:"frequency"`
	Category      string `json:"category"`
	Subcategory   string `json:"subcategory"`
	PaymentMethod string `json:"payment_method"`
	Tags          string `json:"tags"`
	IsPaid        bool   `json:"is_paid"`
	Active        bool   `json:"active"`
}

// BudgetSummary compares a monthly category limit with active expenses.
type BudgetSummary struct {
	UUID           string  `json:"uuid"`
	Month          string  `json:"month"`
	Category       string  `json:"category"`
	LimitCents     int64   `json:"limit_cents"`
	SpentCents     int64   `json:"spent_cents"`
	RemainingCents int64   `json:"remaining_cents"`
	PercentageUsed float64 `json:"percentage_used"`
	OverBudget     bool    `json:"over_budget"`
}
