package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"prisma/internal/models"
	"strings"

	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

// Função privata que encontra o local seguro para salvar o 'prisma.db'
func getDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível encontrar o diretório de config: %w", err)
	}

	// O caminho será .../AppData/Roaming/Prisma/
	appDataDir := filepath.Join(configDir, "Prisma")

	// Garante que o diretório exista
	if err = os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar o diretório de dados: %w", err)
	}

	// Retorna o caminho completo para o arquivo
	return filepath.Join(appDataDir, "prisma.db"), nil
}

func NewRepository() (*Repository, error) {
	dbPath, err := getDatabasePath()
	if err != nil {
		return nil, err
	}

	// sql.Open vai criar o 'prisma.db' se não existir
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir o banco: %w", err)
	}

	// Cria um repositório temporário para chamar a migração
	repo := &Repository{db: db}
	if err = repo.initTables(); err != nil {
		return nil, fmt.Errorf("erro ao inicializar tabelas: %w", err)
	}

	fmt.Println("Banco de dados iniciado com sucesso em: ", dbPath)
	return repo, nil
}

// initTables executa a migração (schema) do banco
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
			active INTEGER NOT NULL DEFAULT 1
		);
	`

	_, err := r.db.Exec(query)
	if err != nil {
		return err
	}

	// Migrate older databases by adding the new columns if they don't exist
	alterQueries := []string{
		"ALTER TABLE transactions ADD COLUMN subcategory TEXT DEFAULT '';",
		"ALTER TABLE transactions ADD COLUMN payment_method TEXT DEFAULT '';",
		"ALTER TABLE transactions ADD COLUMN installments TEXT DEFAULT '';",
		"ALTER TABLE transactions ADD COLUMN tags TEXT DEFAULT '';",
		"ALTER TABLE transactions ADD COLUMN is_paid INTEGER NOT NULL DEFAULT 1;",
	}
	for _, q := range alterQueries {
		r.db.Exec(q) // Ignored if columns already exist
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
	queryBuilder.WriteString("SELECT * FROM transactions WHERE active = 1")

	// args are the values for the '?' placeholders
	var args []interface{}
	
	// Filter by Description (using LIKE)
	if filters.Description != nil && *filters.Description != "" {
		queryBuilder.WriteString(" AND description LIKE ?")
		args = append(args, "%" + *filters.Description + "%")
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
	query := "SELECT * FROM transactions WHERE uuid = ? AND active = 1;"

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
