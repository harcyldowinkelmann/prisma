package models

// StatementInspection describes the columns found in a CSV statement.
type StatementInspection struct {
	Headers                   []string   `json:"headers"`
	SampleRows                [][]string `json:"sample_rows"`
	Delimiter                 string     `json:"delimiter"`
	DetectedDateColumn        int        `json:"detected_date_column"`
	DetectedDescriptionColumn int        `json:"detected_description_column"`
	DetectedAmountColumn      int        `json:"detected_amount_column"`
	DetectedDebitColumn       int        `json:"detected_debit_column"`
	DetectedCreditColumn      int        `json:"detected_credit_column"`
	DetectedDateFormat        string     `json:"detected_date_format"`
}

// StatementParseOptions defines how columns and localized values are parsed.
type StatementParseOptions struct {
	Delimiter         string `json:"delimiter"`
	HasHeader         bool   `json:"has_header"`
	DateColumn        int    `json:"date_column"`
	DescriptionColumn int    `json:"description_column"`
	AmountMode        string `json:"amount_mode"`
	AmountColumn      int    `json:"amount_column"`
	DebitColumn       int    `json:"debit_column"`
	CreditColumn      int    `json:"credit_column"`
	DateFormat        string `json:"date_format"`
	DecimalSeparator  string `json:"decimal_separator"`
}

// StatementEntry is one valid, normalized row from a statement.
type StatementEntry struct {
	RowNumber            int    `json:"row_number"`
	Date                 string `json:"date"`
	Description          string `json:"description"`
	AmountCents          int64  `json:"amount_cents"`
	Type                 int    `json:"type"`
	Occurrence           int    `json:"occurrence"`
	Fingerprint          string `json:"fingerprint"`
	Duplicate            bool   `json:"duplicate"`
	MatchedTransactionID string `json:"matched_transaction_id"`
	MatchedDescription   string `json:"matched_description"`
	MatchedReconciled    bool   `json:"matched_reconciled"`
	Action               string `json:"action"`
}

// StatementRowError identifies a CSV row that could not be normalized.
type StatementRowError struct {
	RowNumber int    `json:"row_number"`
	Message   string `json:"message"`
}

// StatementPreview contains valid statement rows and rejected row details.
type StatementPreview struct {
	Rows   []StatementEntry    `json:"rows"`
	Errors []StatementRowError `json:"errors"`
}

// StatementImportOptions supplies defaults for newly imported transactions.
type StatementImportOptions struct {
	IncomeCategory  string `json:"income_category"`
	ExpenseCategory string `json:"expense_category"`
	Subcategory     string `json:"subcategory"`
	PaymentMethod   string `json:"payment_method"`
	Tags            string `json:"tags"`
}

// StatementImportResult summarizes an atomic statement import.
type StatementImportResult struct {
	ImportedCount   int `json:"imported_count"`
	ReconciledCount int `json:"reconciled_count"`
	SkippedCount    int `json:"skipped_count"`
}
