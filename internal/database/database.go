package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"prisma/internal/models"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

const (
	notificationsEnabledSettingKey = "notifications_enabled"
	currencyCodeSettingKey         = "currency_code"
)

var supportedCurrencyCodes = map[string]bool{
	"AUD": true,
	"BRL": true,
	"CAD": true,
	"EUR": true,
	"GBP": true,
	"JPY": true,
	"USD": true,
}

type columnMigration struct {
	name       string
	definition string
}

var transactionColumnMigrations = []columnMigration{
	{name: "subcategory", definition: "TEXT DEFAULT ''"},
	{name: "payment_method", definition: "TEXT DEFAULT ''"},
	{name: "installments", definition: "TEXT DEFAULT ''"},
	{name: "tags", definition: "TEXT DEFAULT ''"},
	{name: "is_paid", definition: "INTEGER NOT NULL DEFAULT 1"},
	{name: "notified_at", definition: "TEXT DEFAULT ''"},
}

// getDatabasePath returns a safe location for the Prisma database.
func getDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not find the user configuration directory: %w", err)
	}

	// On Windows, this resolves to .../AppData/Roaming/Prisma/.
	appDataDir := filepath.Join(configDir, "Prisma")

	// Ensure that the application data directory exists.
	if err = os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("could not create the application data directory: %w", err)
	}

	// Return the complete path to the database file.
	return filepath.Join(appDataDir, "prisma.db"), nil
}

func NewRepository() (*Repository, error) {
	dbPath, err := getDatabasePath()
	if err != nil {
		return nil, err
	}

	// sql.Open creates prisma.db when it does not exist.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the database: %w", err)
	}

	// Create the repository before applying schema migrations.
	repo := &Repository{db: db}
	if err = repo.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not initialize database tables: %w", err)
	}

	fmt.Println("Database initialized successfully at:", dbPath)
	return repo, nil
}

// initTables creates and migrates the database schema.
func (r *Repository) initTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS transactions (
			uuid TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			date TEXT NOT NULL,
			category TEXT NOT NULL,
			subcategory TEXT DEFAULT '',
			payment_method TEXT DEFAULT '',
			installments TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			is_paid INTEGER NOT NULL DEFAULT 1,
			notified_at TEXT DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1
		);
	`

	_, err := r.db.Exec(query)
	if err != nil {
		return err
	}

	// Create setting tables
	settingsQueries := []string{
		`CREATE TABLE IF NOT EXISTS subcategories (uuid TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, active INTEGER NOT NULL DEFAULT 1);`,
		`CREATE TABLE IF NOT EXISTS payment_methods (uuid TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, active INTEGER NOT NULL DEFAULT 1);`,
		`CREATE TABLE IF NOT EXISTS tags (uuid TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, active INTEGER NOT NULL DEFAULT 1);`,
		`CREATE TABLE IF NOT EXISTS categories (uuid TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, type INTEGER NOT NULL, active INTEGER NOT NULL DEFAULT 1);`,
		`CREATE TABLE IF NOT EXISTS app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
	}
	for _, q := range settingsQueries {
		if _, err := r.db.Exec(q); err != nil {
			return err
		}
	}

	if err := r.ensureTransactionColumns(); err != nil {
		return err
	}
	if err := r.migrateTransactionAmounts(); err != nil {
		return err
	}

	// Seed default categories if none exist
	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		return fmt.Errorf("error counting categories: %w", err)
	}
	if count == 0 {
		seedQuery := "INSERT INTO categories (uuid, name, type, active) VALUES (?, 'Incomes', 1, 1), (?, 'Fixed Expenses', -1, 1), (?, 'Variable Expenses', -1, 1)"
		if _, err := r.db.Exec(seedQuery, uuid.New().String(), uuid.New().String(), uuid.New().String()); err != nil {
			return fmt.Errorf("error seeding default categories: %w", err)
		}
	}

	if _, err := r.db.Exec(
		"INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO NOTHING;",
		notificationsEnabledSettingKey,
		strconv.FormatBool(true),
	); err != nil {
		return fmt.Errorf("error seeding notification settings: %w", err)
	}
	if _, err := r.db.Exec(
		"INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO NOTHING;",
		currencyCodeSettingKey,
		"USD",
	); err != nil {
		return fmt.Errorf("error seeding currency settings: %w", err)
	}

	return nil
}

// GetFinancialMetrics calculates income and expense totals for an inclusive date range.
func (r *Repository) GetFinancialMetrics(startDate string, endDate string) (models.FinancialMetrics, error) {
	metrics := models.FinancialMetrics{Categories: []models.CategoryMetric{}}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return metrics, fmt.Errorf("invalid start date %q: %w", startDate, err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return metrics, fmt.Errorf("invalid end date %q: %w", endDate, err)
	}
	if start.After(end) {
		return metrics, fmt.Errorf("start date must not be after end date")
	}

	rows, err := r.db.Query(`
		SELECT c.name,
		       c.type,
		       COALESCE(SUM(t.amount_cents), 0),
		       COALESCE(SUM(CASE WHEN t.is_paid = 1 THEN t.amount_cents ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN t.is_paid = 0 THEN t.amount_cents ELSE 0 END), 0)
		FROM categories c
		LEFT JOIN transactions t
		       ON t.category = c.name
		      AND t.active = 1
		      AND t.date BETWEEN ? AND ?
		WHERE c.active = 1
		GROUP BY c.uuid, c.name, c.type, c.rowid
		ORDER BY c.rowid ASC;
	`, startDate, endDate)
	if err != nil {
		return metrics, fmt.Errorf("error calculating financial metrics: %w", err)
	}
	defer rows.Close()

	var expectedIncomeCents int64
	var expectedExpensesCents int64
	for rows.Next() {
		var category models.CategoryMetric
		if err := rows.Scan(
			&category.Name,
			&category.Type,
			&category.TotalAmountCents,
			&category.PaidAmountCents,
			&category.PendingAmountCents,
		); err != nil {
			return metrics, fmt.Errorf("error scanning financial metrics: %w", err)
		}

		metrics.Categories = append(metrics.Categories, category)
		switch category.Type {
		case 1:
			expectedIncomeCents += category.TotalAmountCents
			metrics.ReceivedIncomeCents += category.PaidAmountCents
		case -1:
			expectedExpensesCents += category.TotalAmountCents
			metrics.PaidExpensesCents += category.PaidAmountCents
			metrics.PendingExpensesCents += category.PendingAmountCents
		}
	}
	if err := rows.Err(); err != nil {
		return metrics, fmt.Errorf("error iterating financial metrics: %w", err)
	}

	metrics.ActualBalanceCents = metrics.ReceivedIncomeCents - metrics.PaidExpensesCents
	metrics.ExpectedBalanceCents = expectedIncomeCents - expectedExpensesCents
	metrics.HasReceivedIncome = metrics.ReceivedIncomeCents > 0
	if metrics.HasReceivedIncome {
		metrics.IncomeSpentPercentage = float64(metrics.PaidExpensesCents) / float64(metrics.ReceivedIncomeCents) * 100
	}

	return metrics, nil
}

// GetSpendingReport groups active expenses for an inclusive date range.
func (r *Repository) GetSpendingReport(startDate string, endDate string) (models.SpendingReport, error) {
	report := models.SpendingReport{
		StartDate:       startDate,
		EndDate:         endDate,
		ByCategory:      []models.ReportGroup{},
		BySubcategory:   []models.ReportGroup{},
		ByPaymentMethod: []models.ReportGroup{},
		ByTag:           []models.ReportGroup{},
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return report, fmt.Errorf("invalid start date %q: %w", startDate, err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return report, fmt.Errorf("invalid end date %q: %w", endDate, err)
	}
	if start.After(end) {
		return report, fmt.Errorf("start date must not be after end date")
	}

	rows, err := r.db.Query(`
		SELECT t.amount_cents, t.is_paid, t.category, t.subcategory,
		       t.payment_method, t.tags
		FROM transactions t
		INNER JOIN categories c ON c.name = t.category
		WHERE t.active = 1
		  AND c.type = -1
		  AND t.date BETWEEN ? AND ?;
	`, startDate, endDate)
	if err != nil {
		return report, fmt.Errorf("error calculating spending report: %w", err)
	}
	defer rows.Close()

	categoryGroups := make(map[string]*models.ReportGroup)
	subcategoryGroups := make(map[string]*models.ReportGroup)
	paymentMethodGroups := make(map[string]*models.ReportGroup)
	tagGroups := make(map[string]*models.ReportGroup)

	for rows.Next() {
		var (
			amountCents   int64
			isPaid        bool
			category      string
			subcategory   string
			paymentMethod string
			tags          string
		)
		if err := rows.Scan(&amountCents, &isPaid, &category, &subcategory, &paymentMethod, &tags); err != nil {
			return report, fmt.Errorf("error scanning spending report: %w", err)
		}

		if report.TotalExpensesCents, err = addReportCents(report.TotalExpensesCents, amountCents); err != nil {
			return report, err
		}
		if isPaid {
			if report.PaidExpensesCents, err = addReportCents(report.PaidExpensesCents, amountCents); err != nil {
				return report, err
			}
		} else if report.PendingExpensesCents, err = addReportCents(report.PendingExpensesCents, amountCents); err != nil {
			return report, err
		}
		report.TransactionCount++

		if err := addReportTransaction(categoryGroups, reportLabel(category, "Uncategorized"), amountCents, isPaid); err != nil {
			return report, err
		}
		if err := addReportTransaction(subcategoryGroups, reportLabel(subcategory, "Unspecified"), amountCents, isPaid); err != nil {
			return report, err
		}
		if err := addReportTransaction(paymentMethodGroups, reportLabel(paymentMethod, "Unspecified"), amountCents, isPaid); err != nil {
			return report, err
		}

		transactionTags := uniqueReportTags(tags)
		if len(transactionTags) == 0 {
			transactionTags = []string{"Untagged"}
		}
		for _, tag := range transactionTags {
			if err := addReportTransaction(tagGroups, tag, amountCents, isPaid); err != nil {
				return report, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return report, fmt.Errorf("error iterating spending report: %w", err)
	}

	report.ByCategory = finalizeReportGroups(categoryGroups, report.TotalExpensesCents)
	report.BySubcategory = finalizeReportGroups(subcategoryGroups, report.TotalExpensesCents)
	report.ByPaymentMethod = finalizeReportGroups(paymentMethodGroups, report.TotalExpensesCents)
	report.ByTag = finalizeReportGroups(tagGroups, report.TotalExpensesCents)
	return report, nil
}

func addReportCents(current int64, amount int64) (int64, error) {
	if amount > 0 && current > models.MaxSafeAmountCents-amount {
		return 0, fmt.Errorf("spending report total exceeds the maximum exact supported value")
	}
	if amount < 0 && current < -models.MaxSafeAmountCents-amount {
		return 0, fmt.Errorf("spending report total exceeds the maximum exact supported value")
	}
	return current + amount, nil
}

func addReportTransaction(groups map[string]*models.ReportGroup, name string, amountCents int64, isPaid bool) error {
	key := strings.ToLower(name)
	group, exists := groups[key]
	if !exists {
		group = &models.ReportGroup{Name: name}
		groups[key] = group
	}

	var err error
	group.TotalAmountCents, err = addReportCents(group.TotalAmountCents, amountCents)
	if err != nil {
		return err
	}
	if isPaid {
		group.PaidAmountCents, err = addReportCents(group.PaidAmountCents, amountCents)
	} else {
		group.PendingAmountCents, err = addReportCents(group.PendingAmountCents, amountCents)
	}
	if err != nil {
		return err
	}
	group.TransactionCount++
	return nil
}

func reportLabel(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func uniqueReportTags(tags string) []string {
	uniqueTags := make([]string, 0)
	seen := make(map[string]bool)
	for _, tag := range strings.Split(tags, ",") {
		tag = strings.TrimSpace(tag)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] {
			continue
		}
		seen[key] = true
		uniqueTags = append(uniqueTags, tag)
	}
	return uniqueTags
}

func finalizeReportGroups(groups map[string]*models.ReportGroup, totalExpensesCents int64) []models.ReportGroup {
	result := make([]models.ReportGroup, 0, len(groups))
	for _, group := range groups {
		if totalExpensesCents != 0 {
			group.PercentageOfExpenses = float64(group.TotalAmountCents) / float64(totalExpensesCents) * 100
		}
		result = append(result, *group)
	}
	sort.Slice(result, func(i int, j int) bool {
		if result[i].TotalAmountCents == result[j].TotalAmountCents {
			return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
		}
		return result[i].TotalAmountCents > result[j].TotalAmountCents
	})
	return result
}

// ensureTransactionColumns migrates older databases without relying on ignored
// duplicate-column errors. Any unexpected migration failure is returned to the caller.
func (r *Repository) ensureTransactionColumns() error {
	rows, err := r.db.Query("PRAGMA table_info(transactions);")
	if err != nil {
		return fmt.Errorf("error reading transactions schema: %w", err)
	}

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue interface{}
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return fmt.Errorf("error scanning transactions schema: %w", err)
		}
		existingColumns[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("error iterating transactions schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("error closing transactions schema query: %w", err)
	}

	for _, migration := range transactionColumnMigrations {
		if existingColumns[migration.name] {
			continue
		}

		query := fmt.Sprintf("ALTER TABLE transactions ADD COLUMN %s %s;", migration.name, migration.definition)
		if _, err := r.db.Exec(query); err != nil {
			return fmt.Errorf("error adding transactions.%s: %w", migration.name, err)
		}
	}

	return nil
}

// migrateTransactionAmounts replaces the legacy floating-point amount column
// with integer cents while preserving every transaction and its metadata.
func (r *Repository) migrateTransactionAmounts() error {
	columns, err := r.getTransactionColumns()
	if err != nil {
		return err
	}
	if columns["amount_cents"] && !columns["amount"] {
		return nil
	}
	if !columns["amount"] {
		return fmt.Errorf("transactions table has neither amount nor amount_cents")
	}

	amountExpression := "CAST(ROUND(amount * 100.0) AS INTEGER)"
	if columns["amount_cents"] {
		amountExpression = "COALESCE(amount_cents, CAST(ROUND(amount * 100.0) AS INTEGER))"
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting amount migration: %w", err)
	}
	defer tx.Rollback()

	var originalCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM transactions;").Scan(&originalCount); err != nil {
		return fmt.Errorf("error counting transactions before amount migration: %w", err)
	}
	if _, err := tx.Exec("DROP TABLE IF EXISTS transactions_amount_migration;"); err != nil {
		return fmt.Errorf("error clearing stale amount migration table: %w", err)
	}
	if _, err := tx.Exec(`
		CREATE TABLE transactions_amount_migration (
			uuid TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			date TEXT NOT NULL,
			category TEXT NOT NULL,
			subcategory TEXT DEFAULT '',
			payment_method TEXT DEFAULT '',
			installments TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			is_paid INTEGER NOT NULL DEFAULT 1,
			notified_at TEXT DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1
		);
	`); err != nil {
		return fmt.Errorf("error creating amount migration table: %w", err)
	}

	copyQuery := fmt.Sprintf(`
		INSERT INTO transactions_amount_migration (
			uuid, description, amount_cents, date, category, subcategory,
			payment_method, installments, tags, is_paid, notified_at, active
		)
		SELECT uuid, description, %s, date, category, subcategory,
		       payment_method, installments, tags, is_paid, notified_at, active
		FROM transactions;
	`, amountExpression)
	if _, err := tx.Exec(copyQuery); err != nil {
		return fmt.Errorf("error copying transactions during amount migration: %w", err)
	}

	var migratedCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM transactions_amount_migration;").Scan(&migratedCount); err != nil {
		return fmt.Errorf("error counting migrated transactions: %w", err)
	}
	if migratedCount != originalCount {
		return fmt.Errorf("amount migration count mismatch: expected %d transactions, copied %d", originalCount, migratedCount)
	}
	var unsafeAmountCount int
	if err := tx.QueryRow(
		"SELECT COUNT(*) FROM transactions_amount_migration WHERE amount_cents > ? OR amount_cents < ?;",
		models.MaxSafeAmountCents,
		-models.MaxSafeAmountCents,
	).Scan(&unsafeAmountCount); err != nil {
		return fmt.Errorf("error validating migrated transaction amounts: %w", err)
	}
	if unsafeAmountCount != 0 {
		return fmt.Errorf("amount migration found %d values outside the exact supported range", unsafeAmountCount)
	}

	if _, err := tx.Exec("DROP TABLE transactions;"); err != nil {
		return fmt.Errorf("error replacing legacy transactions table: %w", err)
	}
	if _, err := tx.Exec("ALTER TABLE transactions_amount_migration RENAME TO transactions;"); err != nil {
		return fmt.Errorf("error finalizing amount migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing amount migration: %w", err)
	}
	return nil
}

func (r *Repository) getTransactionColumns() (map[string]bool, error) {
	rows, err := r.db.Query("PRAGMA table_info(transactions);")
	if err != nil {
		return nil, fmt.Errorf("error reading transactions schema: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue interface{}
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("error scanning transactions schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions schema: %w", err)
	}
	return columns, nil
}

// Receives transaction data and saves it to the database
func (r *Repository) SaveTransaction(t models.Transaction) error {
	if err := validateAmountCents(t.AmountCents); err != nil {
		return err
	}
	query := `
		INSERT INTO transactions (uuid, description, amount_cents, date, category, subcategory, payment_method, installments, tags, is_paid, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, t.UUID, t.Description, t.AmountCents, t.Date, t.Category, t.Subcategory, t.PaymentMethod, t.Installments, t.Tags, t.IsPaid, t.Active)
	if err != nil {
		return fmt.Errorf("error inserting transaction: %w", err)
	}

	return nil
}

// UpdateTransaction updates an active transaction and resets its reminder state.
func (r *Repository) UpdateTransaction(t models.Transaction) error {
	if err := validateAmountCents(t.AmountCents); err != nil {
		return err
	}
	query := `
		UPDATE transactions
		SET description = ?,
		    amount_cents = ?,
		    date = ?,
		    category = ?,
		    subcategory = ?,
		    payment_method = ?,
		    installments = ?,
		    tags = ?,
		    is_paid = ?,
		    notified_at = ''
		WHERE uuid = ? AND active = 1;
	`

	result, err := r.db.Exec(
		query,
		t.Description,
		t.AmountCents,
		t.Date,
		t.Category,
		t.Subcategory,
		t.PaymentMethod,
		t.Installments,
		t.Tags,
		t.IsPaid,
		t.UUID,
	)
	if err != nil {
		return fmt.Errorf("error updating transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verifying updated transaction: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no active transaction found with UUID: %s", t.UUID)
	}
	return nil
}

func validateAmountCents(amountCents int64) error {
	if amountCents <= 0 {
		return fmt.Errorf("transaction amount must be greater than zero")
	}
	if amountCents > models.MaxSafeAmountCents {
		return fmt.Errorf("transaction amount exceeds the maximum exact supported value")
	}
	return nil
}

// Executes a "soft delete" by updating the 'active' field to 0 (false)
func (r *Repository) SoftDeleteTransaction(uuid string) error {
	query := `
		UPDATE transactions
		SET active = 0
		WHERE uuid = ?;
	`

	// Execute SQL, passing UUID as argument.
	res, err := r.db.Exec(query, uuid)
	if err != nil {
		return fmt.Errorf("error executing soft delete: %w", err)
	}

	// Verifying if any record was affected, confirming SoftDelete
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verifying affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no transaction found with the provided UUID: %s", uuid)
	}

	return nil
}

// RestoreTransaction makes an archived transaction active again.
func (r *Repository) RestoreTransaction(uuid string) error {
	result, err := r.db.Exec(
		"UPDATE transactions SET active = 1, notified_at = '' WHERE uuid = ? AND active = 0;",
		uuid,
	)
	if err != nil {
		return fmt.Errorf("error restoring transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verifying restored transaction: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no archived transaction found with UUID: %s", uuid)
	}
	return nil
}

// Fetches active transactions based on optional filters
func (r *Repository) GetTransactions(filters models.TransactionFilters) ([]models.Transaction, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT uuid, description, amount_cents, date, category, subcategory, payment_method, installments, tags, is_paid, active FROM transactions WHERE 1 = 1")

	// args are the values for the '?' placeholders
	var args []interface{}

	if !filters.IncludeArchived {
		queryBuilder.WriteString(" AND active = 1")
	}

	// Filter by Description (using LIKE)
	if filters.Description != nil && *filters.Description != "" {
		queryBuilder.WriteString(" AND description LIKE ?")
		args = append(args, "%"+*filters.Description+"%")
	}

	// Filter by Amount (exact)
	if filters.AmountCents != nil {
		queryBuilder.WriteString(" AND amount_cents = ?")
		args = append(args, *filters.AmountCents)
	}

	// Filter by Date (exact)
	if filters.Date != nil && *filters.Date != "" {
		queryBuilder.WriteString(" AND date = ?")
		args = append(args, *filters.Date)
	}
	if filters.StartDate != nil && *filters.StartDate != "" {
		queryBuilder.WriteString(" AND date >= ?")
		args = append(args, *filters.StartDate)
	}
	if filters.EndDate != nil && *filters.EndDate != "" {
		queryBuilder.WriteString(" AND date <= ?")
		args = append(args, *filters.EndDate)
	}

	// Filter by Category (exact)
	if filters.Category != nil && *filters.Category != "" {
		queryBuilder.WriteString(" AND category = ?")
		args = append(args, *filters.Category)
	}
	if filters.IsPaid != nil {
		queryBuilder.WriteString(" AND is_paid = ?")
		args = append(args, *filters.IsPaid)
	}

	queryBuilder.WriteString(" ORDER BY date DESC, description ASC;")

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error executing dynamic search: %w", err)
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

// Fetches an active Transaction by its UUID
func (r *Repository) GetTransactionByID(uuid string) (models.Transaction, error) {
	query := "SELECT uuid, description, amount_cents, date, category, subcategory, payment_method, installments, tags, is_paid, active FROM transactions WHERE uuid = ? AND active = 1;"

	var t models.Transaction

	err := r.db.QueryRow(query, uuid).Scan(&t.UUID, &t.Description, &t.AmountCents, &t.Date, &t.Category, &t.Subcategory, &t.PaymentMethod, &t.Installments, &t.Tags, &t.IsPaid, &t.Active)
	if err != nil {
		if err == sql.ErrNoRows {
			return t, fmt.Errorf("no active transaction found with the UUID: %s", uuid)
		}
		return t, fmt.Errorf("error searching by UUID: %w", err)
	}

	return t, nil
}

// --- HELPER FUNCTIONS ---

// scanTransactions (Helper)
func (r *Repository) scanTransactions(rows *sql.Rows) ([]models.Transaction, error) {
	var transactions []models.Transaction

	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.UUID, &t.Description, &t.AmountCents, &t.Date, &t.Category, &t.Subcategory, &t.PaymentMethod, &t.Installments, &t.Tags, &t.IsPaid, &t.Active); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in result rows: %w", err)
	}

	return transactions, nil
}

// --- NOTIFICATIONS ---

// GetPendingNotifications returns unpaid transactions due today (or overdue) that haven't been notified today.
func (r *Repository) GetPendingNotifications(todayDate string) ([]models.Transaction, error) {
	query := `
		SELECT t.uuid, t.description, t.amount_cents, t.date, t.category, t.subcategory,
		       t.payment_method, t.installments, t.tags, t.is_paid, t.active
		FROM transactions t
		INNER JOIN categories c ON c.name = t.category
		WHERE t.active = 1
		  AND t.is_paid = 0
		  AND t.date <= ?
		  AND COALESCE(t.notified_at, '') != ?
		  AND c.active = 1
		  AND c.type = -1
	`

	rows, err := r.db.Query(query, todayDate, todayDate)
	if err != nil {
		return nil, fmt.Errorf("error fetching pending notifications: %w", err)
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

// MarkAsNotified updates the notified_at field so we don't notify again on the same day.
func (r *Repository) MarkAsNotified(uuid string, date string) error {
	query := "UPDATE transactions SET notified_at = ? WHERE uuid = ?"
	result, err := r.db.Exec(query, date, uuid)
	if err != nil {
		return fmt.Errorf("error marking transaction as notified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking notified transaction: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no transaction found with UUID: %s", uuid)
	}

	return nil
}

// GetNotificationsEnabled returns the persisted payment reminder preference.
func (r *Repository) GetNotificationsEnabled() (bool, error) {
	var value string
	if err := r.db.QueryRow(
		"SELECT value FROM app_settings WHERE key = ?;",
		notificationsEnabledSettingKey,
	).Scan(&value); err != nil {
		return false, fmt.Errorf("error reading notification settings: %w", err)
	}

	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid notification setting %q: %w", value, err)
	}

	return enabled, nil
}

// SetNotificationsEnabled persists whether payment reminders are enabled.
func (r *Repository) SetNotificationsEnabled(enabled bool) error {
	_, err := r.db.Exec(
		`INSERT INTO app_settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value;`,
		notificationsEnabledSettingKey,
		strconv.FormatBool(enabled),
	)
	if err != nil {
		return fmt.Errorf("error updating notification settings: %w", err)
	}

	return nil
}

// GetCurrencyCode returns the ISO currency code used to format monetary values.
func (r *Repository) GetCurrencyCode() (string, error) {
	var currencyCode string
	if err := r.db.QueryRow(
		"SELECT value FROM app_settings WHERE key = ?;",
		currencyCodeSettingKey,
	).Scan(&currencyCode); err != nil {
		return "", fmt.Errorf("error reading currency settings: %w", err)
	}
	if !supportedCurrencyCodes[currencyCode] {
		return "", fmt.Errorf("unsupported stored currency code: %s", currencyCode)
	}
	return currencyCode, nil
}

// SetCurrencyCode persists the ISO currency code used by the interface.
func (r *Repository) SetCurrencyCode(currencyCode string) error {
	currencyCode = strings.ToUpper(strings.TrimSpace(currencyCode))
	if !supportedCurrencyCodes[currencyCode] {
		return fmt.Errorf("unsupported currency code: %s", currencyCode)
	}

	_, err := r.db.Exec(
		`INSERT INTO app_settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value;`,
		currencyCodeSettingKey,
		currencyCode,
	)
	if err != nil {
		return fmt.Errorf("error updating currency settings: %w", err)
	}
	return nil
}

// --- SETTINGS CRUD ---

func (r *Repository) GetSettings(tableName string) ([]models.SettingItem, error) {
	allowedTables := map[string]bool{"subcategories": true, "payment_methods": true, "tags": true}
	if !allowedTables[tableName] {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	query := fmt.Sprintf("SELECT uuid, name, active FROM %s WHERE active = 1 ORDER BY name ASC;", tableName)
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", tableName, err)
	}
	defer rows.Close()

	var items []models.SettingItem
	for rows.Next() {
		var item models.SettingItem
		var activeInt int
		if err := rows.Scan(&item.UUID, &item.Name, &activeInt); err != nil {
			return nil, fmt.Errorf("error scanning row in %s: %w", tableName, err)
		}
		item.Active = activeInt == 1
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in result rows for %s: %w", tableName, err)
	}

	return items, nil
}

func (r *Repository) SaveSetting(tableName string, item models.SettingItem) error {
	allowedTables := map[string]bool{"subcategories": true, "payment_methods": true, "tags": true}
	if !allowedTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	activeInt := 0
	if item.Active {
		activeInt = 1
	}

	query := fmt.Sprintf("INSERT INTO %s (uuid, name, active) VALUES (?, ?, ?);", tableName)
	_, err := r.db.Exec(query, item.UUID, item.Name, activeInt)
	if err != nil {
		return fmt.Errorf("error inserting into %s: %w", tableName, err)
	}
	return nil
}

func (r *Repository) UpdateSetting(tableName string, item models.SettingItem) error {
	allowedTables := map[string]bool{"subcategories": true, "payment_methods": true, "tags": true}
	if !allowedTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	query := fmt.Sprintf("UPDATE %s SET name = ? WHERE uuid = ?;", tableName)
	res, err := r.db.Exec(query, item.Name, item.UUID)
	if err != nil {
		return fmt.Errorf("error updating %s: %w", tableName, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("error verifying affected rows or no item found")
	}
	return nil
}

func (r *Repository) SoftDeleteSetting(tableName string, uuid string) error {
	allowedTables := map[string]bool{"subcategories": true, "payment_methods": true, "tags": true}
	if !allowedTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	query := fmt.Sprintf("UPDATE %s SET active = 0 WHERE uuid = ?;", tableName)
	res, err := r.db.Exec(query, uuid)
	if err != nil {
		return fmt.Errorf("error executing soft delete on %s: %w", tableName, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("error verifying affected rows or no item found")
	}
	return nil
}

// --- CATEGORIES CRUD ---

// AddCategory adds a new dynamic column category
func (r *Repository) AddCategory(name string, t int) error {
	id := uuid.New().String()
	query := "INSERT INTO categories (uuid, name, type, active) VALUES (?, ?, ?, 1)"
	_, err := r.db.Exec(query, id, name, t)
	return err
}

// GetCategories returns all active categories
func (r *Repository) GetCategories() ([]models.Category, error) {
	query := "SELECT uuid, name, type, active FROM categories WHERE active = 1 ORDER BY ROWID ASC"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.UUID, &c.Name, &c.Type, &c.Active); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// UpdateCategory updates category
func (r *Repository) UpdateCategory(uuid string, name string, t int) error {
	query := "UPDATE categories SET name = ?, type = ? WHERE uuid = ?"
	_, err := r.db.Exec(query, name, t, uuid)
	return err
}

// SoftDeleteCategory
func (r *Repository) SoftDeleteCategory(uuid string) error {
	query := "UPDATE categories SET active = 0 WHERE uuid = ?"
	_, err := r.db.Exec(query, uuid)
	return err
}
