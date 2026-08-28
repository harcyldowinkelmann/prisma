package models

// ReportGroup summarizes expense transactions that share one report dimension.
type ReportGroup struct {
	Name                 string  `json:"name"`
	TotalAmountCents     int64   `json:"total_amount_cents"`
	PaidAmountCents      int64   `json:"paid_amount_cents"`
	PendingAmountCents   int64   `json:"pending_amount_cents"`
	TransactionCount     int     `json:"transaction_count"`
	PercentageOfExpenses float64 `json:"percentage_of_expenses"`
}

// SpendingReport summarizes active expenses for an inclusive date range.
type SpendingReport struct {
	StartDate            string        `json:"start_date"`
	EndDate              string        `json:"end_date"`
	TotalExpensesCents   int64         `json:"total_expenses_cents"`
	PaidExpensesCents    int64         `json:"paid_expenses_cents"`
	PendingExpensesCents int64         `json:"pending_expenses_cents"`
	TransactionCount     int           `json:"transaction_count"`
	ByCategory           []ReportGroup `json:"by_category"`
	BySubcategory        []ReportGroup `json:"by_subcategory"`
	ByPaymentMethod      []ReportGroup `json:"by_payment_method"`
	ByTag                []ReportGroup `json:"by_tag"`
}
