package main

import (
	"context"
	"fmt"
	"os"
	"prisma/internal/database"
	"prisma/internal/models"
	"prisma/internal/notifier"
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
// TODO: Validate incoming data before passing it to the save function
func (a *App) SaveTransaction(description string, amount float64, date string, category string, subcategory string, paymentMethod string, installments string, tags string, isPaid bool) (string, error) {
	newUUID := uuid.New()

	// 2. Create the data model
	newTransaction := models.Transaction{
		UUID:          newUUID,
		Description:   description,
		Amount:        amount,
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
	err := a.db.SaveTransaction(newTransaction)
	if err != nil {
		// Return error to Vue.js
		return "", fmt.Errorf("error saving to database: %w", err)
	}

	// Return a success string and 'nil' for error
	return "Transaction saved successfully!", nil
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
