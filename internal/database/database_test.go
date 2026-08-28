package database

import (
	"database/sql"
	"math"
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

func TestGetFinancialMetricsCalculatesSelectedPeriod(t *testing.T) {
	repo := newTestRepository(t)
	if err := repo.AddCategory("Subscriptions", -1); err != nil {
		t.Fatalf("add custom expense category: %v", err)
	}

	transactions := []models.Transaction{
		{UUID: uuid.New(), Description: "Received salary", Amount: 3000, Date: "2026-08-05", Category: "Incomes", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Expected bonus", Amount: 500, Date: "2026-08-20", Category: "Incomes", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "Paid rent", Amount: 1200, Date: "2026-08-10", Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending electricity", Amount: 300, Date: "2026-08-25", Category: "Fixed Expenses", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "Paid groceries", Amount: 200, Date: "2026-08-11", Category: "Variable Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending subscription", Amount: 100, Date: "2026-08-12", Category: "Subscriptions", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "Archived expense", Amount: 999, Date: "2026-08-15", Category: "Variable Expenses", IsPaid: true, Active: false},
		{UUID: uuid.New(), Description: "Previous month expense", Amount: 800, Date: "2026-07-31", Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Next month income", Amount: 900, Date: "2026-09-01", Category: "Incomes", IsPaid: true, Active: true},
	}
	for _, transaction := range transactions {
		saveTestTransaction(t, repo, transaction)
	}

	metrics, err := repo.GetFinancialMetrics("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("get financial metrics: %v", err)
	}

	assertFloatEqual(t, "received income", metrics.ReceivedIncome, 3000)
	assertFloatEqual(t, "paid expenses", metrics.PaidExpenses, 1400)
	assertFloatEqual(t, "pending expenses", metrics.PendingExpenses, 400)
	assertFloatEqual(t, "actual balance", metrics.ActualBalance, 1600)
	assertFloatEqual(t, "expected balance", metrics.ExpectedBalance, 1700)
	assertFloatEqual(t, "income spent percentage", metrics.IncomeSpentPercentage, 46.6666666667)
	if !metrics.HasReceivedIncome {
		t.Fatal("expected received income flag to be true")
	}

	wantCategories := map[string]models.CategoryMetric{
		"Incomes":           {TotalAmount: 3500, PaidAmount: 3000, PendingAmount: 500},
		"Fixed Expenses":    {TotalAmount: 1500, PaidAmount: 1200, PendingAmount: 300},
		"Variable Expenses": {TotalAmount: 200, PaidAmount: 200, PendingAmount: 0},
		"Subscriptions":     {TotalAmount: 100, PaidAmount: 0, PendingAmount: 100},
	}
	if len(metrics.Categories) != len(wantCategories) {
		t.Fatalf("expected %d category metrics, got %d", len(wantCategories), len(metrics.Categories))
	}
	for _, category := range metrics.Categories {
		want, ok := wantCategories[category.Name]
		if !ok {
			t.Errorf("unexpected category metric %q", category.Name)
			continue
		}
		assertFloatEqual(t, category.Name+" total", category.TotalAmount, want.TotalAmount)
		assertFloatEqual(t, category.Name+" paid", category.PaidAmount, want.PaidAmount)
		assertFloatEqual(t, category.Name+" pending", category.PendingAmount, want.PendingAmount)
	}
}

func TestGetFinancialMetricsHandlesNoReceivedIncome(t *testing.T) {
	repo := newTestRepository(t)
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Pending expense", Amount: 50,
		Date: "2026-08-10", Category: "Fixed Expenses", Active: true,
	})

	metrics, err := repo.GetFinancialMetrics("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("get financial metrics without income: %v", err)
	}
	if metrics.HasReceivedIncome {
		t.Fatal("expected received income flag to be false")
	}
	assertFloatEqual(t, "income spent percentage", metrics.IncomeSpentPercentage, 0)
	assertFloatEqual(t, "expected balance", metrics.ExpectedBalance, -50)
}

func TestGetFinancialMetricsValidatesDateRange(t *testing.T) {
	repo := newTestRepository(t)
	testCases := []struct {
		name      string
		startDate string
		endDate   string
	}{
		{name: "invalid start", startDate: "08/01/2026", endDate: "2026-08-31"},
		{name: "invalid end", startDate: "2026-08-01", endDate: "August 31"},
		{name: "reversed range", startDate: "2026-08-31", endDate: "2026-08-01"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := repo.GetFinancialMetrics(testCase.startDate, testCase.endDate); err == nil {
				t.Fatal("expected an invalid date range error")
			}
		})
	}
}

func TestCurrencySettingPersistsAndRejectsUnsupportedCodes(t *testing.T) {
	repo := newTestRepository(t)

	currencyCode, err := repo.GetCurrencyCode()
	if err != nil {
		t.Fatalf("get default currency: %v", err)
	}
	if currencyCode != "USD" {
		t.Fatalf("expected default currency USD, got %s", currencyCode)
	}

	if err := repo.SetCurrencyCode(" brl "); err != nil {
		t.Fatalf("set supported currency: %v", err)
	}
	if err := repo.initTables(); err != nil {
		t.Fatalf("repeat schema initialization: %v", err)
	}
	currencyCode, err = repo.GetCurrencyCode()
	if err != nil {
		t.Fatalf("get persisted currency: %v", err)
	}
	if currencyCode != "BRL" {
		t.Fatalf("expected persisted currency BRL, got %s", currencyCode)
	}

	if err := repo.SetCurrencyCode("XYZ"); err == nil {
		t.Fatal("expected unsupported currency to be rejected")
	}
}

func assertFloatEqual(t *testing.T, label string, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Errorf("%s: expected %.10f, got %.10f", label, want, got)
	}
}
