package main

import (
	"context"
	"fmt"
	"os"
	"prisma/internal/database"
	"prisma/internal/models"
	"prisma/internal/notifier"
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

// Fetches an active Transaction by its UUID
func (a *App) GetTransactionByID(uuid string) (models.Transaction, error) {
	return a.db.GetTransactionByID(uuid)
}

// GetTransactions is the bridge for dynamic search.
// Ex: { "description": "market", "category": "Variable Expenses" }
func (a *App) GetTransactions(filters models.TransactionFilters) ([]models.Transaction, error) {
	return a.db.GetTransactions(filters)
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
