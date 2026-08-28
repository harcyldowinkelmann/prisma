package models

// CategoryMetric contains the totals for one active transaction category.
type CategoryMetric struct {
	Name               string `json:"name"`
	Type               int    `json:"type"`
	TotalAmountCents   int64  `json:"total_amount_cents"`
	PaidAmountCents    int64  `json:"paid_amount_cents"`
	PendingAmountCents int64  `json:"pending_amount_cents"`
}

// FinancialMetrics contains the calculated totals for a date range.
type FinancialMetrics struct {
	ReceivedIncomeCents   int64            `json:"received_income_cents"`
	PaidExpensesCents     int64            `json:"paid_expenses_cents"`
	PendingExpensesCents  int64            `json:"pending_expenses_cents"`
	ActualBalanceCents    int64            `json:"actual_balance_cents"`
	ExpectedBalanceCents  int64            `json:"expected_balance_cents"`
	IncomeSpentPercentage float64          `json:"income_spent_percentage"`
	HasReceivedIncome     bool             `json:"has_received_income"`
	Categories            []CategoryMetric `json:"categories"`
}
