package models

// CategoryMetric contains the totals for one active transaction category.
type CategoryMetric struct {
	Name          string  `json:"name"`
	Type          int     `json:"type"`
	TotalAmount   float64 `json:"total_amount"`
	PaidAmount    float64 `json:"paid_amount"`
	PendingAmount float64 `json:"pending_amount"`
}

// FinancialMetrics contains the calculated totals for a date range.
type FinancialMetrics struct {
	ReceivedIncome        float64          `json:"received_income"`
	PaidExpenses          float64          `json:"paid_expenses"`
	PendingExpenses       float64          `json:"pending_expenses"`
	ActualBalance         float64          `json:"actual_balance"`
	ExpectedBalance       float64          `json:"expected_balance"`
	IncomeSpentPercentage float64          `json:"income_spent_percentage"`
	HasReceivedIncome     bool             `json:"has_received_income"`
	Categories            []CategoryMetric `json:"categories"`
}
