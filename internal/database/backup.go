package database

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"prisma/internal/models"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildBackup reads every user-owned data table into a portable snapshot.
func (r *Repository) BuildBackup(createdAt time.Time) (models.BackupData, error) {
	backup := models.BackupData{
		FormatVersion:      models.BackupFormatVersion,
		CreatedAt:          createdAt.UTC().Format(time.RFC3339),
		Transactions:       []models.BackupTransaction{},
		Categories:         []models.Category{},
		Subcategories:      []models.SettingItem{},
		PaymentMethods:     []models.SettingItem{},
		Tags:               []models.SettingItem{},
		RecurringSchedules: []models.RecurringSchedule{},
		Budgets:            []models.BackupBudget{},
		AppSettings:        []models.BackupSetting{},
	}
	tx, err := r.db.Begin()
	if err != nil {
		return backup, fmt.Errorf("error starting backup snapshot: %w", err)
	}
	defer tx.Rollback()

	transactionRows, err := tx.Query(`
		SELECT uuid, description, amount_cents, date, category, subcategory,
		       payment_method, installments, tags, is_paid, reconciled,
		       import_fingerprint, recurrence_id, occurrence_date,
		       installment_group, notified_at, active
		FROM transactions
		ORDER BY date, uuid;
	`)
	if err != nil {
		return backup, fmt.Errorf("error reading transactions for backup: %w", err)
	}
	for transactionRows.Next() {
		var transaction models.BackupTransaction
		if err := transactionRows.Scan(
			&transaction.UUID, &transaction.Description, &transaction.AmountCents,
			&transaction.Date, &transaction.Category, &transaction.Subcategory,
			&transaction.PaymentMethod, &transaction.Installments, &transaction.Tags,
			&transaction.IsPaid, &transaction.Reconciled, &transaction.ImportFingerprint,
			&transaction.RecurrenceID, &transaction.OccurrenceDate,
			&transaction.InstallmentGroup, &transaction.NotifiedAt, &transaction.Active,
		); err != nil {
			transactionRows.Close()
			return backup, fmt.Errorf("error scanning transaction backup: %w", err)
		}
		backup.Transactions = append(backup.Transactions, transaction)
	}
	if err := closeRows(transactionRows, "transaction backup"); err != nil {
		return backup, err
	}

	categoryRows, err := tx.Query("SELECT uuid, name, type, active FROM categories ORDER BY rowid;")
	if err != nil {
		return backup, fmt.Errorf("error reading categories for backup: %w", err)
	}
	for categoryRows.Next() {
		var category models.Category
		if err := categoryRows.Scan(&category.UUID, &category.Name, &category.Type, &category.Active); err != nil {
			categoryRows.Close()
			return backup, fmt.Errorf("error scanning category backup: %w", err)
		}
		backup.Categories = append(backup.Categories, category)
	}
	if err := closeRows(categoryRows, "category backup"); err != nil {
		return backup, err
	}

	if backup.Subcategories, err = getAllBackupItems(tx, "subcategories"); err != nil {
		return backup, err
	}
	if backup.PaymentMethods, err = getAllBackupItems(tx, "payment_methods"); err != nil {
		return backup, err
	}
	if backup.Tags, err = getAllBackupItems(tx, "tags"); err != nil {
		return backup, err
	}

	recurringRows, err := tx.Query(`
		SELECT uuid, description, amount_cents, start_date, end_date, frequency,
		       category, subcategory, payment_method, tags, is_paid, active
		FROM recurring_schedules ORDER BY start_date, uuid;
	`)
	if err != nil {
		return backup, fmt.Errorf("error reading recurring schedules for backup: %w", err)
	}
	for recurringRows.Next() {
		var schedule models.RecurringSchedule
		if err := recurringRows.Scan(
			&schedule.UUID, &schedule.Description, &schedule.AmountCents,
			&schedule.StartDate, &schedule.EndDate, &schedule.Frequency,
			&schedule.Category, &schedule.Subcategory, &schedule.PaymentMethod,
			&schedule.Tags, &schedule.IsPaid, &schedule.Active,
		); err != nil {
			recurringRows.Close()
			return backup, fmt.Errorf("error scanning recurring schedule backup: %w", err)
		}
		backup.RecurringSchedules = append(backup.RecurringSchedules, schedule)
	}
	if err := closeRows(recurringRows, "recurring schedule backup"); err != nil {
		return backup, err
	}

	budgetRows, err := tx.Query("SELECT uuid, month, category, limit_cents FROM budgets ORDER BY month, category;")
	if err != nil {
		return backup, fmt.Errorf("error reading budgets for backup: %w", err)
	}
	for budgetRows.Next() {
		var budget models.BackupBudget
		if err := budgetRows.Scan(&budget.UUID, &budget.Month, &budget.Category, &budget.LimitCents); err != nil {
			budgetRows.Close()
			return backup, fmt.Errorf("error scanning budget backup: %w", err)
		}
		backup.Budgets = append(backup.Budgets, budget)
	}
	if err := closeRows(budgetRows, "budget backup"); err != nil {
		return backup, err
	}

	settingRows, err := tx.Query("SELECT key, value FROM app_settings ORDER BY key;")
	if err != nil {
		return backup, fmt.Errorf("error reading application settings for backup: %w", err)
	}
	for settingRows.Next() {
		var setting models.BackupSetting
		if err := settingRows.Scan(&setting.Key, &setting.Value); err != nil {
			settingRows.Close()
			return backup, fmt.Errorf("error scanning application setting backup: %w", err)
		}
		backup.AppSettings = append(backup.AppSettings, setting)
	}
	if err := closeRows(settingRows, "application setting backup"); err != nil {
		return backup, err
	}

	if err := tx.Commit(); err != nil {
		return backup, fmt.Errorf("error finalizing backup snapshot: %w", err)
	}
	return backup, nil
}

// RestoreBackup validates the complete snapshot before replacing data atomically.
func (r *Repository) RestoreBackup(backup models.BackupData) error {
	if err := validateBackup(backup); err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting backup restore: %w", err)
	}
	defer tx.Rollback()

	for _, table := range []string{
		"transactions", "budgets", "recurring_schedules", "subcategories",
		"payment_methods", "tags", "categories", "app_settings",
	} {
		if _, err := tx.Exec("DELETE FROM " + table + ";"); err != nil {
			return fmt.Errorf("error clearing %s during restore: %w", table, err)
		}
	}

	for _, category := range backup.Categories {
		if _, err := tx.Exec(
			"INSERT INTO categories (uuid, name, type, active) VALUES (?, ?, ?, ?);",
			category.UUID, category.Name, category.Type, category.Active,
		); err != nil {
			return fmt.Errorf("error restoring category %q: %w", category.Name, err)
		}
	}
	for tableName, items := range map[string][]models.SettingItem{
		"subcategories":   backup.Subcategories,
		"payment_methods": backup.PaymentMethods,
		"tags":            backup.Tags,
	} {
		for _, item := range items {
			query := fmt.Sprintf("INSERT INTO %s (uuid, name, active) VALUES (?, ?, ?);", tableName)
			if _, err := tx.Exec(query, item.UUID, item.Name, item.Active); err != nil {
				return fmt.Errorf("error restoring %s item %q: %w", tableName, item.Name, err)
			}
		}
	}
	for _, schedule := range backup.RecurringSchedules {
		if _, err := tx.Exec(`
			INSERT INTO recurring_schedules (
				uuid, description, amount_cents, start_date, end_date, frequency,
				category, subcategory, payment_method, tags, is_paid, active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, schedule.UUID, schedule.Description, schedule.AmountCents,
			schedule.StartDate, schedule.EndDate, schedule.Frequency,
			schedule.Category, schedule.Subcategory, schedule.PaymentMethod,
			schedule.Tags, schedule.IsPaid, schedule.Active); err != nil {
			return fmt.Errorf("error restoring recurring schedule %q: %w", schedule.Description, err)
		}
	}
	for _, budget := range backup.Budgets {
		if _, err := tx.Exec(
			"INSERT INTO budgets (uuid, month, category, limit_cents) VALUES (?, ?, ?, ?);",
			budget.UUID, budget.Month, budget.Category, budget.LimitCents,
		); err != nil {
			return fmt.Errorf("error restoring budget for %q: %w", budget.Category, err)
		}
	}
	for _, transaction := range backup.Transactions {
		if _, err := tx.Exec(`
			INSERT INTO transactions (
				uuid, description, amount_cents, date, category, subcategory,
				payment_method, installments, tags, is_paid, reconciled,
				import_fingerprint, recurrence_id, occurrence_date,
				installment_group, notified_at, active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, transaction.UUID, transaction.Description, transaction.AmountCents,
			transaction.Date, transaction.Category, transaction.Subcategory,
			transaction.PaymentMethod, transaction.Installments, transaction.Tags,
			transaction.IsPaid, transaction.Reconciled, transaction.ImportFingerprint,
			transaction.RecurrenceID, transaction.OccurrenceDate,
			transaction.InstallmentGroup, transaction.NotifiedAt, transaction.Active); err != nil {
			return fmt.Errorf("error restoring transaction %q: %w", transaction.Description, err)
		}
	}
	for _, setting := range backup.AppSettings {
		if _, err := tx.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?);", setting.Key, setting.Value); err != nil {
			return fmt.Errorf("error restoring application setting %q: %w", setting.Key, err)
		}
	}
	if _, err := tx.Exec(
		"INSERT INTO app_settings (key, value) VALUES (?, 'true') ON CONFLICT(key) DO NOTHING;",
		notificationsEnabledSettingKey,
	); err != nil {
		return fmt.Errorf("error restoring default notification setting: %w", err)
	}
	if _, err := tx.Exec(
		"INSERT INTO app_settings (key, value) VALUES (?, 'USD') ON CONFLICT(key) DO NOTHING;",
		currencyCodeSettingKey,
	); err != nil {
		return fmt.Errorf("error restoring default currency setting: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing backup restore: %w", err)
	}
	return nil
}

// BuildTransactionsCSV exports all active and archived transactions for spreadsheets.
func (r *Repository) BuildTransactionsCSV() ([]byte, error) {
	backup, err := r.BuildBackup(time.Now())
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	buffer.WriteString("\ufeff")
	writer := csv.NewWriter(&buffer)
	headers := []string{
		"ID", "Description", "Amount", "Date", "Category", "Subcategory",
		"Payment Method", "Installments", "Tags", "Payment Status",
		"Reconciliation Status", "Record Status",
	}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("error writing CSV header: %w", err)
	}
	for _, transaction := range backup.Transactions {
		paymentStatus := "Pending"
		if transaction.IsPaid {
			paymentStatus = "Paid"
		}
		reconciliationStatus := "Unreconciled"
		if transaction.Reconciled {
			reconciliationStatus = "Reconciled"
		}
		recordStatus := "Archived"
		if transaction.Active {
			recordStatus = "Active"
		}
		record := []string{
			transaction.UUID, transaction.Description, formatBackupCents(transaction.AmountCents),
			transaction.Date, transaction.Category, transaction.Subcategory,
			transaction.PaymentMethod, transaction.Installments, transaction.Tags,
			paymentStatus, reconciliationStatus, recordStatus,
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("error writing transaction CSV: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error finalizing transaction CSV: %w", err)
	}
	return buffer.Bytes(), nil
}

type backupRowsQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func getAllBackupItems(queryer backupRowsQueryer, tableName string) ([]models.SettingItem, error) {
	allowedTables := map[string]bool{"subcategories": true, "payment_methods": true, "tags": true}
	if !allowedTables[tableName] {
		return nil, fmt.Errorf("invalid backup table: %s", tableName)
	}
	rows, err := queryer.Query(fmt.Sprintf("SELECT uuid, name, active FROM %s ORDER BY name, uuid;", tableName))
	if err != nil {
		return nil, fmt.Errorf("error reading %s for backup: %w", tableName, err)
	}
	items := []models.SettingItem{}
	for rows.Next() {
		var item models.SettingItem
		if err := rows.Scan(&item.UUID, &item.Name, &item.Active); err != nil {
			rows.Close()
			return nil, fmt.Errorf("error scanning %s backup: %w", tableName, err)
		}
		items = append(items, item)
	}
	if err := closeRows(rows, tableName+" backup"); err != nil {
		return nil, err
	}
	return items, nil
}

type rowsCloser interface {
	Err() error
	Close() error
}

func closeRows(rows rowsCloser, label string) error {
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("error iterating %s: %w", label, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("error closing %s: %w", label, err)
	}
	return nil
}

func validateBackup(backup models.BackupData) error {
	if backup.FormatVersion != models.BackupFormatVersion {
		return fmt.Errorf("unsupported backup format version: %d", backup.FormatVersion)
	}
	if _, err := time.Parse(time.RFC3339, backup.CreatedAt); err != nil {
		return fmt.Errorf("backup creation timestamp is invalid")
	}
	if len(backup.Categories) == 0 {
		return fmt.Errorf("backup must contain at least one category")
	}

	categoryTypes := make(map[string]int)
	categoryIDs := make(map[string]bool)
	for _, category := range backup.Categories {
		if err := validateBackupUUID(category.UUID, categoryIDs, "category"); err != nil {
			return err
		}
		name := strings.TrimSpace(category.Name)
		if name == "" || categoryTypes[name] != 0 {
			return fmt.Errorf("backup contains an empty or duplicate category name")
		}
		if category.Type != 1 && category.Type != -1 {
			return fmt.Errorf("category %q has an invalid type", category.Name)
		}
		categoryTypes[name] = category.Type
	}

	if err := validateBackupItems(backup.Subcategories, "subcategory"); err != nil {
		return err
	}
	if err := validateBackupItems(backup.PaymentMethods, "payment method"); err != nil {
		return err
	}
	if err := validateBackupItems(backup.Tags, "tag"); err != nil {
		return err
	}

	transactionIDs := make(map[string]bool)
	fingerprints := make(map[string]bool)
	recurrenceOccurrences := make(map[string]bool)
	for _, transaction := range backup.Transactions {
		if err := validateBackupUUID(transaction.UUID, transactionIDs, "transaction"); err != nil {
			return err
		}
		if strings.TrimSpace(transaction.Description) == "" {
			return fmt.Errorf("backup contains a transaction without a description")
		}
		if err := validateAmountCents(transaction.AmountCents); err != nil {
			return fmt.Errorf("transaction %q has an invalid amount: %w", transaction.Description, err)
		}
		if _, err := time.Parse("2006-01-02", transaction.Date); err != nil {
			return fmt.Errorf("transaction %q has an invalid date", transaction.Description)
		}
		if categoryTypes[transaction.Category] == 0 {
			return fmt.Errorf("transaction %q references an unknown category", transaction.Description)
		}
		if transaction.ImportFingerprint != "" {
			if fingerprints[transaction.ImportFingerprint] {
				return fmt.Errorf("backup contains duplicate statement fingerprints")
			}
			fingerprints[transaction.ImportFingerprint] = true
		}
		if transaction.RecurrenceID != "" {
			if transaction.OccurrenceDate == "" {
				return fmt.Errorf("recurring transaction %q has no occurrence date", transaction.Description)
			}
			if _, err := time.Parse("2006-01-02", transaction.OccurrenceDate); err != nil {
				return fmt.Errorf("recurring transaction %q has an invalid occurrence date", transaction.Description)
			}
			key := transaction.RecurrenceID + "\x00" + transaction.OccurrenceDate
			if recurrenceOccurrences[key] {
				return fmt.Errorf("backup contains duplicate recurring occurrences")
			}
			recurrenceOccurrences[key] = true
		}
		if transaction.NotifiedAt != "" {
			if _, err := time.Parse("2006-01-02", transaction.NotifiedAt); err != nil {
				return fmt.Errorf("transaction %q has an invalid notification date", transaction.Description)
			}
		}
	}

	scheduleIDs := make(map[string]bool)
	for _, schedule := range backup.RecurringSchedules {
		if err := validateBackupUUID(schedule.UUID, scheduleIDs, "recurring schedule"); err != nil {
			return err
		}
		if strings.TrimSpace(schedule.Description) == "" {
			return fmt.Errorf("backup contains a recurring schedule without a description")
		}
		if err := validateAmountCents(schedule.AmountCents); err != nil {
			return fmt.Errorf("recurring schedule %q has an invalid amount: %w", schedule.Description, err)
		}
		start, err := time.Parse("2006-01-02", schedule.StartDate)
		if err != nil {
			return fmt.Errorf("recurring schedule %q has an invalid start date", schedule.Description)
		}
		if schedule.EndDate != "" {
			end, err := time.Parse("2006-01-02", schedule.EndDate)
			if err != nil || end.Before(start) {
				return fmt.Errorf("recurring schedule %q has an invalid end date", schedule.Description)
			}
		}
		if schedule.Frequency != "weekly" && schedule.Frequency != "monthly" && schedule.Frequency != "yearly" {
			return fmt.Errorf("recurring schedule %q has an invalid frequency", schedule.Description)
		}
		if categoryTypes[schedule.Category] == 0 {
			return fmt.Errorf("recurring schedule %q references an unknown category", schedule.Description)
		}
	}

	budgetIDs := make(map[string]bool)
	budgetKeys := make(map[string]bool)
	for _, budget := range backup.Budgets {
		if err := validateBackupUUID(budget.UUID, budgetIDs, "budget"); err != nil {
			return err
		}
		if _, err := time.Parse("2006-01", budget.Month); err != nil {
			return fmt.Errorf("budget for %q has an invalid month", budget.Category)
		}
		if err := validateAmountCents(budget.LimitCents); err != nil {
			return fmt.Errorf("budget for %q has an invalid limit: %w", budget.Category, err)
		}
		if categoryTypes[budget.Category] != -1 {
			return fmt.Errorf("budget references a category that is not an expense")
		}
		key := budget.Month + "\x00" + budget.Category
		if budgetKeys[key] {
			return fmt.Errorf("backup contains duplicate monthly category budgets")
		}
		budgetKeys[key] = true
	}

	settingKeys := make(map[string]bool)
	for _, setting := range backup.AppSettings {
		if strings.TrimSpace(setting.Key) == "" || settingKeys[setting.Key] {
			return fmt.Errorf("backup contains an empty or duplicate application setting")
		}
		settingKeys[setting.Key] = true
		switch setting.Key {
		case currencyCodeSettingKey:
			if !supportedCurrencyCodes[setting.Value] {
				return fmt.Errorf("backup contains an unsupported currency code")
			}
		case notificationsEnabledSettingKey:
			if _, err := strconv.ParseBool(setting.Value); err != nil {
				return fmt.Errorf("backup contains an invalid notification setting")
			}
		}
	}
	return nil
}

func validateBackupItems(items []models.SettingItem, label string) error {
	ids := make(map[string]bool)
	names := make(map[string]bool)
	for _, item := range items {
		if err := validateBackupUUID(item.UUID, ids, label); err != nil {
			return err
		}
		name := strings.TrimSpace(item.Name)
		if name == "" || names[name] {
			return fmt.Errorf("backup contains an empty or duplicate %s name", label)
		}
		names[name] = true
	}
	return nil
}

func validateBackupUUID(value string, seen map[string]bool, label string) error {
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("backup contains an invalid %s UUID", label)
	}
	if seen[value] {
		return fmt.Errorf("backup contains a duplicate %s UUID", label)
	}
	seen[value] = true
	return nil
}

func formatBackupCents(amountCents int64) string {
	sign := ""
	if amountCents < 0 {
		sign = "-"
		amountCents = -amountCents
	}
	return fmt.Sprintf("%s%d.%02d", sign, amountCents/100, amountCents%100)
}
