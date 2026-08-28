package database

import (
	"database/sql"
	"path/filepath"
	"prisma/internal/models"
	"testing"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "prisma-test.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	repo := &Repository{db: db}
	if err := repo.initTables(); err != nil {
		t.Fatalf("initialize test database: %v", err)
	}

	return repo
}

func saveTestTransaction(t *testing.T, repo *Repository, transaction models.Transaction) {
	t.Helper()
	if err := repo.SaveTransaction(transaction); err != nil {
		t.Fatalf("save transaction %q: %v", transaction.Description, err)
	}
}

func TestInitTablesMigratesLegacySchemaAndPreservesPreference(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy-prisma.db"))
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close legacy database: %v", err)
		}
	})

	_, err = db.Exec(`CREATE TABLE transactions (
		uuid TEXT PRIMARY KEY,
		description TEXT NOT NULL,
		amount REAL NOT NULL,
		date TEXT NOT NULL,
		category TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1
	);`)
	if err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	repo := &Repository{db: db}
	if err := repo.initTables(); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	if err := repo.initTables(); err != nil {
		t.Fatalf("repeat schema migration: %v", err)
	}

	columns := make(map[string]bool)
	rows, err := db.Query("PRAGMA table_info(transactions);")
	if err != nil {
		t.Fatalf("read migrated schema: %v", err)
	}
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
			t.Fatalf("scan migrated schema: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatalf("iterate migrated schema: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close migrated schema query: %v", err)
	}

	for _, migration := range transactionColumnMigrations {
		if !columns[migration.name] {
			t.Errorf("expected migrated column %q", migration.name)
		}
	}

	enabled, err := repo.GetNotificationsEnabled()
	if err != nil {
		t.Fatalf("read default notification preference: %v", err)
	}
	if !enabled {
		t.Fatal("expected notifications to be enabled by default")
	}

	if err := repo.SetNotificationsEnabled(false); err != nil {
		t.Fatalf("disable notifications: %v", err)
	}
	if err := repo.initTables(); err != nil {
		t.Fatalf("repeat migration after preference update: %v", err)
	}

	enabled, err = repo.GetNotificationsEnabled()
	if err != nil {
		t.Fatalf("read persisted notification preference: %v", err)
	}
	if enabled {
		t.Fatal("expected schema initialization to preserve the disabled preference")
	}
}

func TestGetPendingNotificationsFiltersIneligibleTransactions(t *testing.T) {
	repo := newTestRepository(t)
	const today = "2026-08-26"

	eligible := []models.Transaction{
		{UUID: uuid.New(), Description: "Due today", Amount: 100, Date: today, Category: "Fixed Expenses", Active: true},
		{UUID: uuid.New(), Description: "Overdue", Amount: 200, Date: "2026-08-25", Category: "Variable Expenses", Active: true},
		{UUID: uuid.New(), Description: "Notified yesterday", Amount: 300, Date: "2026-08-24", Category: "Fixed Expenses", Active: true},
	}
	for _, transaction := range eligible {
		saveTestTransaction(t, repo, transaction)
	}
	if err := repo.MarkAsNotified(eligible[2].UUID.String(), "2026-08-25"); err != nil {
		t.Fatalf("mark transaction as notified yesterday: %v", err)
	}

	ineligible := []models.Transaction{
		{UUID: uuid.New(), Description: "Future expense", Amount: 10, Date: "2026-08-27", Category: "Fixed Expenses", Active: true},
		{UUID: uuid.New(), Description: "Paid expense", Amount: 20, Date: today, Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending income", Amount: 30, Date: today, Category: "Incomes", Active: true},
		{UUID: uuid.New(), Description: "Archived expense", Amount: 40, Date: today, Category: "Fixed Expenses", Active: false},
		{UUID: uuid.New(), Description: "Already notified", Amount: 50, Date: today, Category: "Fixed Expenses", Active: true},
	}
	for _, transaction := range ineligible {
		saveTestTransaction(t, repo, transaction)
	}
	if err := repo.MarkAsNotified(ineligible[4].UUID.String(), today); err != nil {
		t.Fatalf("mark transaction as notified today: %v", err)
	}

	if err := repo.AddCategory("Inactive Expenses", -1); err != nil {
		t.Fatalf("add inactive expense category: %v", err)
	}
	categories, err := repo.GetCategories()
	if err != nil {
		t.Fatalf("get categories: %v", err)
	}
	for _, category := range categories {
		if category.Name == "Inactive Expenses" {
			if err := repo.SoftDeleteCategory(category.UUID); err != nil {
				t.Fatalf("inactivate expense category: %v", err)
			}
		}
	}
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Inactive category", Amount: 60,
		Date: today, Category: "Inactive Expenses", Active: true,
	})

	pending, err := repo.GetPendingNotifications(today)
	if err != nil {
		t.Fatalf("get pending notifications: %v", err)
	}

	want := map[string]bool{
		"Due today":          true,
		"Overdue":            true,
		"Notified yesterday": true,
	}
	if len(pending) != len(want) {
		t.Fatalf("expected %d pending expenses, got %d: %#v", len(want), len(pending), pending)
	}
	for _, transaction := range pending {
		if !want[transaction.Description] {
			t.Errorf("unexpected pending transaction %q", transaction.Description)
		}
	}
}

func TestMarkAsNotifiedRejectsUnknownTransaction(t *testing.T) {
	repo := newTestRepository(t)
	if err := repo.MarkAsNotified(uuid.New().String(), "2026-08-26"); err == nil {
		t.Fatal("expected an error for an unknown transaction")
	}
}

func TestGetNotificationsEnabledRejectsInvalidStoredValue(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.db.Exec(
		"UPDATE app_settings SET value = ? WHERE key = ?;",
		"invalid",
		notificationsEnabledSettingKey,
	)
	if err != nil {
		t.Fatalf("store invalid notification preference: %v", err)
	}

	if _, err := repo.GetNotificationsEnabled(); err == nil {
		t.Fatal("expected an error for an invalid notification preference")
	}
}
