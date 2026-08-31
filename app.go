package main

import (
	"context"
	"fmt"
	"os"
	"prisma/internal/database"
	"prisma/internal/models"
	"prisma/internal/notifier"
	"prisma/internal/statement"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// App struct
type App struct {
	ctx            context.Context
	db             *database.Repository
	notifierCancel context.CancelFunc
	notifierDone   <-chan struct{}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initializes the database repository
	repo, err := database.NewRepository()
	if err != nil {
		// TODO: Instead of Println, use Wails Runtime
		// to show a fatal error dialog.
		fmt.Printf("FATAL ERROR INITIALIZING DATABASE: %v\n", err)
		os.Exit(1)
	}

	// Injects the repository into the App struct
	a.db = repo
	if _, err := repo.GenerateRecurringTransactions(lastDayOfCurrentMonth()); err != nil {
		fmt.Printf("ERROR GENERATING RECURRING TRANSACTIONS: %v\n", err)
	}

	// Start the background notification service
	notifierCtx, cancelNotifier := context.WithCancel(ctx)
	a.notifierCancel = cancelNotifier
	a.notifierDone = notifier.StartBackgroundNotifier(notifierCtx, repo)
}

// shutdown stops background services before the application exits.
func (a *App) shutdown(_ context.Context) {
	if a.notifierCancel != nil {
		a.notifierCancel()
	}
	if a.notifierDone != nil {
		select {
		case <-a.notifierDone:
		case <-time.After(2 * time.Second):
			fmt.Println("Timed out while stopping the background notification service.")
		}
	}
}

// SaveTransaction is the bridge function the Vue frontend will call.
func (a *App) SaveTransaction(description string, amount string, date string, category string, subcategory string, paymentMethod string, installments string, tags string, isPaid bool) (string, error) {
	amountCents, err := parseAmountToCents(amount)
	if err != nil {
		return "", err
	}
	newUUID := uuid.New()

	// 2. Create the data model
	newTransaction := models.Transaction{
		UUID:          newUUID,
		Description:   description,
		AmountCents:   amountCents,
		Date:          date,
		Category:      category,
		Subcategory:   subcategory,
		PaymentMethod: paymentMethod,
		Installments:  installments,
		Tags:          tags,
		IsPaid:        isPaid,
		Active:        true,
	}

	// 3. Call the "backend" layer (Repository)
	err = a.db.SaveTransaction(newTransaction)
	if err != nil {
		// Return error to Vue.js
		return "", fmt.Errorf("error saving to database: %w", err)
	}

	// Return a success string and 'nil' for error
	return "Transaction saved successfully!", nil
}

// SaveInstallmentTransactions creates one transaction or an exact monthly installment plan.
func (a *App) SaveInstallmentTransactions(description string, amount string, date string, category string, subcategory string, paymentMethod string, tags string, isPaid bool, installmentCount int) (string, error) {
	amountCents, err := parseAmountToCents(amount)
	if err != nil {
		return "", err
	}
	transaction := models.Transaction{
		Description: strings.TrimSpace(description), AmountCents: amountCents, Date: date,
		Category: category, Subcategory: subcategory, PaymentMethod: paymentMethod,
		Tags: tags, IsPaid: isPaid, Active: true,
	}
	if err := a.db.SaveInstallmentTransactions(transaction, installmentCount); err != nil {
		return "", err
	}
	return "Transaction plan saved successfully!", nil
}

// UpdateTransaction updates an existing active transaction.
func (a *App) UpdateTransaction(transactionUUID string, description string, amount string, date string, category string, subcategory string, paymentMethod string, installments string, tags string, isPaid bool) (string, error) {
	parsedUUID, err := uuid.Parse(transactionUUID)
	if err != nil {
		return "", fmt.Errorf("invalid transaction UUID: %w", err)
	}
	amountCents, err := parseAmountToCents(amount)
	if err != nil {
		return "", err
	}

	transaction := models.Transaction{
		UUID:          parsedUUID,
		Description:   description,
		AmountCents:   amountCents,
		Date:          date,
		Category:      category,
		Subcategory:   subcategory,
		PaymentMethod: paymentMethod,
		Installments:  installments,
		Tags:          tags,
		IsPaid:        isPaid,
		Active:        true,
	}

	if err := a.db.UpdateTransaction(transaction); err != nil {
		return "", fmt.Errorf("error updating transaction: %w", err)
	}
	return "Transaction updated successfully!", nil
}

// parseAmountToCents converts a user-entered decimal string without using
// floating-point arithmetic, preventing rounding errors at the data boundary.
func parseAmountToCents(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	parts := strings.Split(amount, ".")
	if len(parts) > 2 || amount == "" {
		return 0, fmt.Errorf("amount must be a positive number with at most two decimal places")
	}

	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}
	fractionPart := ""
	if len(parts) == 2 {
		fractionPart = parts[1]
	}
	if len(fractionPart) > 2 || !containsOnlyDigits(wholePart) || !containsOnlyDigits(fractionPart) {
		return 0, fmt.Errorf("amount must be a positive number with at most two decimal places")
	}

	fractionPart += strings.Repeat("0", 2-len(fractionPart))
	wholePart = strings.TrimLeft(wholePart, "0")
	if wholePart == "" {
		wholePart = "0"
	}

	cents, err := strconv.ParseInt(wholePart+fractionPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount is too large: %w", err)
	}
	if cents <= 0 {
		return 0, fmt.Errorf("amount must be greater than zero")
	}
	if cents > models.MaxSafeAmountCents {
		return 0, fmt.Errorf("amount exceeds the maximum supported value")
	}
	return cents, nil
}

func containsOnlyDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func lastDayOfCurrentMonth() string {
	now := time.Now()
	return time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
}

func (a *App) SoftDeleteTransaction(uuid string) (string, error) {
	if uuid == "" {
		return "", fmt.Errorf("provided UUID is empty")
	}

	err := a.db.SoftDeleteTransaction(uuid)
	if err != nil {
		return "", err
	}

	return "Transaction archived successfully!", nil
}

// RestoreTransaction restores an archived transaction.
func (a *App) RestoreTransaction(uuid string) (string, error) {
	if uuid == "" {
		return "", fmt.Errorf("provided UUID is empty")
	}
	if err := a.db.RestoreTransaction(uuid); err != nil {
		return "", err
	}
	return "Transaction restored successfully!", nil
}

// SetTransactionReconciled changes whether a transaction matches a bank statement.
func (a *App) SetTransactionReconciled(transactionUUID string, reconciled bool) error {
	if _, err := uuid.Parse(transactionUUID); err != nil {
		return fmt.Errorf("invalid transaction UUID: %w", err)
	}
	return a.db.SetTransactionReconciled(transactionUUID, reconciled)
}

// --- SETTINGS CRUD BRIDGE ---

func (a *App) GetSettings(tableName string) ([]models.SettingItem, error) {
	return a.db.GetSettings(tableName)
}

func (a *App) AddSetting(tableName string, name string) error {
	newUUID := uuid.New().String()
	item := models.SettingItem{
		UUID:   newUUID,
		Name:   name,
		Active: true,
	}
	return a.db.SaveSetting(tableName, item)
}

func (a *App) UpdateSetting(tableName string, uuid string, newName string) error {
	item := models.SettingItem{
		UUID: uuid,
		Name: newName,
	}
	return a.db.UpdateSetting(tableName, item)
}

func (a *App) InactivateSetting(tableName string, uuid string) error {
	return a.db.SoftDeleteSetting(tableName, uuid)
}

// GetNotificationsEnabled returns whether payment reminders are enabled.
func (a *App) GetNotificationsEnabled() (bool, error) {
	return a.db.GetNotificationsEnabled()
}

// SetNotificationsEnabled persists the payment reminder preference.
func (a *App) SetNotificationsEnabled(enabled bool) error {
	return a.db.SetNotificationsEnabled(enabled)
}

// GetCurrencyCode returns the currency selected for monetary values.
func (a *App) GetCurrencyCode() (string, error) {
	return a.db.GetCurrencyCode()
}

// SetCurrencyCode persists the currency selected for monetary values.
func (a *App) SetCurrencyCode(currencyCode string) error {
	return a.db.SetCurrencyCode(currencyCode)
}

// GetFinancialMetrics returns calculated totals for an inclusive date range.
func (a *App) GetFinancialMetrics(startDate string, endDate string) (models.FinancialMetrics, error) {
	return a.db.GetFinancialMetrics(startDate, endDate)
}

// GetSpendingReport returns expense breakdowns for an inclusive date range.
func (a *App) GetSpendingReport(startDate string, endDate string) (models.SpendingReport, error) {
	return a.db.GetSpendingReport(startDate, endDate)
}

// Fetches an active Transaction by its UUID
func (a *App) GetTransactionByID(uuid string) (models.Transaction, error) {
	return a.db.GetTransactionByID(uuid)
}

// GetTransactions is the bridge for dynamic search.
// Ex: { "description": "market", "category": "Variable Expenses" }
func (a *App) GetTransactions(filters models.TransactionFilters) ([]models.Transaction, error) {
	return a.db.GetTransactions(filters)
}

// InspectStatementCSV returns columns and detected mappings without changing data.
func (a *App) InspectStatementCSV(content string, delimiter string, hasHeader bool) (models.StatementInspection, error) {
	return statement.Inspect(content, delimiter, hasHeader)
}

// PreviewStatementCSV parses rows and identifies duplicates or reconciliation matches.
func (a *App) PreviewStatementCSV(content string, options models.StatementParseOptions) (models.StatementPreview, error) {
	parsed, err := statement.Parse(content, options)
	if err != nil {
		return parsed, err
	}
	prepared, err := a.db.PrepareStatementPreview(parsed.Rows)
	if err != nil {
		return parsed, err
	}
	prepared.Errors = parsed.Errors
	return prepared, nil
}

// ImportStatementRows applies the user-confirmed preview atomically.
func (a *App) ImportStatementRows(entries []models.StatementEntry, options models.StatementImportOptions) (models.StatementImportResult, error) {
	return a.db.ImportStatementRows(entries, options)
}

// AddRecurringSchedule stores a rule and generates its occurrences through the current month.
func (a *App) AddRecurringSchedule(description string, amount string, startDate string, endDate string, frequency string, category string, subcategory string, paymentMethod string, tags string, isPaid bool) error {
	amountCents, err := parseAmountToCents(amount)
	if err != nil {
		return err
	}
	schedule := models.RecurringSchedule{
		UUID: uuid.New().String(), Description: description, AmountCents: amountCents,
		StartDate: startDate, EndDate: endDate, Frequency: frequency, Category: category,
		Subcategory: subcategory, PaymentMethod: paymentMethod, Tags: tags, IsPaid: isPaid, Active: true,
	}
	if err := a.db.SaveRecurringSchedule(schedule); err != nil {
		return err
	}
	if _, err = a.db.GenerateRecurringTransactions(lastDayOfCurrentMonth()); err != nil {
		_ = a.db.SoftDeleteRecurringSchedule(schedule.UUID)
		return err
	}
	return nil
}

func (a *App) GetRecurringSchedules() ([]models.RecurringSchedule, error) {
	return a.db.GetRecurringSchedules()
}

func (a *App) StopRecurringSchedule(scheduleUUID string) error {
	if _, err := uuid.Parse(scheduleUUID); err != nil {
		return fmt.Errorf("invalid recurring schedule UUID: %w", err)
	}
	return a.db.SoftDeleteRecurringSchedule(scheduleUUID)
}

func (a *App) GenerateRecurringTransactions(throughDate string) (int, error) {
	return a.db.GenerateRecurringTransactions(throughDate)
}

func (a *App) SaveBudget(month string, category string, amount string) error {
	amountCents, err := parseAmountToCents(amount)
	if err != nil {
		return err
	}
	return a.db.SaveBudget(month, category, amountCents)
}

func (a *App) DeleteBudget(month string, category string) error {
	return a.db.DeleteBudget(month, category)
}

func (a *App) GetBudgetSummaries(month string) ([]models.BudgetSummary, error) {
	return a.db.GetBudgetSummaries(month)
}

// --- CATEGORIES BRIDGE ---

func (a *App) AddCategory(name string, t int) error {
	return a.db.AddCategory(name, t)
}

func (a *App) GetCategories() ([]models.Category, error) {
	return a.db.GetCategories()
}

func (a *App) UpdateCategory(uuid string, name string, t int) error {
	return a.db.UpdateCategory(uuid, name, t)
}

func (a *App) SoftDeleteCategory(uuid string) error {
	return a.db.SoftDeleteCategory(uuid)
}
