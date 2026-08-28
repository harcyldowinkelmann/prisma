package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"prisma/internal/models"
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
			amount REAL NOT NULL,
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
		       COALESCE(SUM(t.amount), 0),
		       COALESCE(SUM(CASE WHEN t.is_paid = 1 THEN t.amount ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN t.is_paid = 0 THEN t.amount ELSE 0 END), 0)
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

	var expectedIncome float64
	var expectedExpenses float64
	for rows.Next() {
		var category models.CategoryMetric
		if err := rows.Scan(
			&category.Name,
			&category.Type,
			&category.TotalAmount,
			&category.PaidAmount,
			&category.PendingAmount,
		); err != nil {
			return metrics, fmt.Errorf("error scanning financial metrics: %w", err)
		}

		metrics.Categories = append(metrics.Categories, category)
		switch category.Type {
		case 1:
			expectedIncome += category.TotalAmount
			metrics.ReceivedIncome += category.PaidAmount
		case -1:
			expectedExpenses += category.TotalAmount
			metrics.PaidExpenses += category.PaidAmount
			metrics.PendingExpenses += category.PendingAmount
		}
	}
	if err := rows.Err(); err != nil {
		return metrics, fmt.Errorf("error iterating financial metrics: %w", err)
	}

	metrics.ActualBalance = metrics.ReceivedIncome - metrics.PaidExpenses
	metrics.ExpectedBalance = expectedIncome - expectedExpenses
	metrics.HasReceivedIncome = metrics.ReceivedIncome > 0
	if metrics.HasReceivedIncome {
		metrics.IncomeSpentPercentage = metrics.PaidExpenses / metrics.ReceivedIncome * 100
	}

	return metrics, nil
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

// Receives transaction data and saves it to the database
func (r *Repository) SaveTransaction(t models.Transaction) error {
	query := `
		INSERT INTO transactions (uuid, description, amount, date, category, subcategory, payment_method, installments, tags, is_paid, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, t.UUID, t.Description, t.Amount, t.Date, t.Category, t.Subcategory, t.PaymentMethod, t.Installments, t.Tags, t.IsPaid, t.Active)
	if err != nil {
		return fmt.Errorf("error inserting transaction: %w", err)
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

// Fetches active transactions based on optional filters
func (r *Repository) GetTransactions(filters models.TransactionFilters) ([]models.Transaction, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT uuid, description, amount, date, category, subcategory, payment_method, installments, tags, is_paid, active FROM transactions WHERE active = 1")

	// args are the values for the '?' placeholders
	var args []interface{}

	// Filter by Description (using LIKE)
	if filters.Description != nil && *filters.Description != "" {
		queryBuilder.WriteString(" AND description LIKE ?")
		args = append(args, "%"+*filters.Description+"%")
	}

	// Filter by Amount (exact)
	if filters.Amount != nil {
		queryBuilder.WriteString(" AND amount = ?")
		args = append(args, *filters.Amount)
	}

	// Filter by Date (exact)
	if filters.Date != nil && *filters.Date != "" {
		queryBuilder.WriteString(" AND date = ?")
		args = append(args, *filters.Date)
	}

	// Filter by Category (exact)
	if filters.Category != nil && *filters.Category != "" {
		queryBuilder.WriteString(" AND category = ?")
		args = append(args, *filters.Category)
	}

	queryBuilder.WriteString(" ORDER BY date DESC;")

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error executing dynamic search: %w", err)
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

// Fetches an active Transaction by its UUID
func (r *Repository) GetTransactionByID(uuid string) (models.Transaction, error) {
	query := "SELECT uuid, description, amount, date, category, subcategory, payment_method, installments, tags, is_paid, active FROM transactions WHERE uuid = ? AND active = 1;"

	var t models.Transaction

	err := r.db.QueryRow(query, uuid).Scan(&t.UUID, &t.Description, &t.Amount, &t.Date, &t.Category, &t.Subcategory, &t.PaymentMethod, &t.Installments, &t.Tags, &t.IsPaid, &t.Active)
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
		if err := rows.Scan(&t.UUID, &t.Description, &t.Amount, &t.Date, &t.Category, &t.Subcategory, &t.PaymentMethod, &t.Installments, &t.Tags, &t.IsPaid, &t.Active); err != nil {
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
		SELECT t.uuid, t.description, t.amount, t.date, t.category, t.subcategory,
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
