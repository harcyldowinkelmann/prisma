package main

import (
	"context"
	"fmt"
	"os"
	"prisma/internal/database"
	"prisma/internal/models"

	"github.com/google/uuid"
)

// App struct
type App struct {
	ctx context.Context
	db	*database.Repository
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
}

// SaveTransaction is the bridge function the Vue frontend will call.
// TODO: Validate incoming data before passing it to the save function
func (a *App) SaveTransaction(description string, amount float64, date string, category string) (string, error) {
	newUUID := uuid.New()

	// 2. Create the data model
	newTransaction := models.Transaction{
		UUID:        newUUID,
		Description: description,
		Amount:      amount,
		Date:        date,
		Category:    category,
		Active:      true,
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

// Bridge function the Vue frontend will call to perform Soft Delete
// Receives UUID (as string) and passes it to the repository
func (a *App) SoftDeleteTransaction(uuid string) (string, error) {
	if uuid == "" {
		return "", fmt.Errorf("provided UUID is empty")
	}

	// Call the "backend" layer (Repository)
	err := a.db.SoftDeleteTransaction(uuid)
	if err != nil {
		return "", err
	}

	// Return a success string and 'nil' for error
	return "Transaction archived successfully!", nil
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
