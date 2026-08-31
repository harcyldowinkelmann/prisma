package models

// BackupFormatVersion is the current portable Prisma backup schema.
const BackupFormatVersion = 1

// BackupTransaction preserves a transaction and its internal provenance fields.
type BackupTransaction struct {
	UUID              string `json:"uuid"`
	Description       string `json:"description"`
	AmountCents       int64  `json:"amount_cents"`
	Date              string `json:"date"`
	Category          string `json:"category"`
	Subcategory       string `json:"subcategory"`
	PaymentMethod     string `json:"payment_method"`
	Installments      string `json:"installments"`
	Tags              string `json:"tags"`
	IsPaid            bool   `json:"is_paid"`
	Reconciled        bool   `json:"reconciled"`
	ImportFingerprint string `json:"import_fingerprint"`
	RecurrenceID      string `json:"recurrence_id"`
	OccurrenceDate    string `json:"occurrence_date"`
	InstallmentGroup  string `json:"installment_group"`
	NotifiedAt        string `json:"notified_at"`
	Active            bool   `json:"active"`
}

// BackupBudget preserves a category limit without calculated fields.
type BackupBudget struct {
	UUID       string `json:"uuid"`
	Month      string `json:"month"`
	Category   string `json:"category"`
	LimitCents int64  `json:"limit_cents"`
}

// BackupSetting preserves a generic application preference.
type BackupSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// BackupData is a complete, versioned, portable Prisma data snapshot.
type BackupData struct {
	FormatVersion      int                 `json:"format_version"`
	CreatedAt          string              `json:"created_at"`
	Transactions       []BackupTransaction `json:"transactions"`
	Categories         []Category          `json:"categories"`
	Subcategories      []SettingItem       `json:"subcategories"`
	PaymentMethods     []SettingItem       `json:"payment_methods"`
	Tags               []SettingItem       `json:"tags"`
	RecurringSchedules []RecurringSchedule `json:"recurring_schedules"`
	Budgets            []BackupBudget      `json:"budgets"`
	AppSettings        []BackupSetting     `json:"app_settings"`
}
