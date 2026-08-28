package database

import (
	"database/sql"
	"math"
	"path/filepath"
	"prisma/internal/models"
	"strings"
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
	_, err = db.Exec(`
		INSERT INTO transactions (uuid, description, amount, date, category, active)
		VALUES (?, 'Exact decimal', 12.34, '2026-08-01', 'Fixed Expenses', 1),
		       (?, 'Single decimal', 0.1, '2026-08-02', 'Fixed Expenses', 1),
		       (?, 'Rounded legacy decimal', 19.995, '2026-08-03', 'Fixed Expenses', 1);
	`, uuid.New().String(), uuid.New().String(), uuid.New().String())
	if err != nil {
		t.Fatalf("insert legacy transactions: %v", err)
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
	if !columns["amount_cents"] {
		t.Fatal("expected integer amount_cents column after migration")
	}
	if columns["amount"] {
		t.Fatal("expected legacy floating-point amount column to be removed")
	}

	wantCents := map[string]int64{
		"Exact decimal":          1234,
		"Single decimal":         10,
		"Rounded legacy decimal": 2000,
	}
	amountRows, err := db.Query("SELECT description, amount_cents FROM transactions;")
	if err != nil {
		t.Fatalf("read migrated transaction amounts: %v", err)
	}
	for amountRows.Next() {
		var description string
		var amountCents int64
		if err := amountRows.Scan(&description, &amountCents); err != nil {
			amountRows.Close()
			t.Fatalf("scan migrated transaction amount: %v", err)
		}
		if amountCents != wantCents[description] {
			t.Errorf("%s: expected %d cents, got %d", description, wantCents[description], amountCents)
		}
		delete(wantCents, description)
	}
	if err := amountRows.Err(); err != nil {
		amountRows.Close()
		t.Fatalf("iterate migrated transaction amounts: %v", err)
	}
	if err := amountRows.Close(); err != nil {
		t.Fatalf("close migrated transaction amounts: %v", err)
	}
	if len(wantCents) != 0 {
		t.Fatalf("missing migrated transaction amounts: %#v", wantCents)
	}
	var nonIntegerAmounts int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE typeof(amount_cents) != 'integer';",
	).Scan(&nonIntegerAmounts); err != nil {
		t.Fatalf("verify migrated amount storage type: %v", err)
	}
	if nonIntegerAmounts != 0 {
		t.Fatalf("expected every migrated amount to use SQLite integer storage, found %d invalid values", nonIntegerAmounts)
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
		{UUID: uuid.New(), Description: "Due today", AmountCents: 10000, Date: today, Category: "Fixed Expenses", Active: true},
		{UUID: uuid.New(), Description: "Overdue", AmountCents: 20000, Date: "2026-08-25", Category: "Variable Expenses", Active: true},
		{UUID: uuid.New(), Description: "Notified yesterday", AmountCents: 30000, Date: "2026-08-24", Category: "Fixed Expenses", Active: true},
	}
	for _, transaction := range eligible {
		saveTestTransaction(t, repo, transaction)
	}
	if err := repo.MarkAsNotified(eligible[2].UUID.String(), "2026-08-25"); err != nil {
		t.Fatalf("mark transaction as notified yesterday: %v", err)
	}

	ineligible := []models.Transaction{
		{UUID: uuid.New(), Description: "Future expense", AmountCents: 1000, Date: "2026-08-27", Category: "Fixed Expenses", Active: true},
		{UUID: uuid.New(), Description: "Paid expense", AmountCents: 2000, Date: today, Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending income", AmountCents: 3000, Date: today, Category: "Incomes", Active: true},
		{UUID: uuid.New(), Description: "Archived expense", AmountCents: 4000, Date: today, Category: "Fixed Expenses", Active: false},
		{UUID: uuid.New(), Description: "Already notified", AmountCents: 5000, Date: today, Category: "Fixed Expenses", Active: true},
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
		UUID: uuid.New(), Description: "Inactive category", AmountCents: 6000,
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
		{UUID: uuid.New(), Description: "Received salary", AmountCents: 300000, Date: "2026-08-05", Category: "Incomes", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Expected bonus", AmountCents: 50000, Date: "2026-08-20", Category: "Incomes", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "Paid rent", AmountCents: 120000, Date: "2026-08-10", Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending electricity", AmountCents: 30000, Date: "2026-08-25", Category: "Fixed Expenses", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "Paid groceries", AmountCents: 20000, Date: "2026-08-11", Category: "Variable Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending subscription", AmountCents: 10000, Date: "2026-08-12", Category: "Subscriptions", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "Archived expense", AmountCents: 99900, Date: "2026-08-15", Category: "Variable Expenses", IsPaid: true, Active: false},
		{UUID: uuid.New(), Description: "Previous month expense", AmountCents: 80000, Date: "2026-07-31", Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Next month income", AmountCents: 90000, Date: "2026-09-01", Category: "Incomes", IsPaid: true, Active: true},
	}
	for _, transaction := range transactions {
		saveTestTransaction(t, repo, transaction)
	}

	metrics, err := repo.GetFinancialMetrics("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("get financial metrics: %v", err)
	}

	assertInt64Equal(t, "received income", metrics.ReceivedIncomeCents, 300000)
	assertInt64Equal(t, "paid expenses", metrics.PaidExpensesCents, 140000)
	assertInt64Equal(t, "pending expenses", metrics.PendingExpensesCents, 40000)
	assertInt64Equal(t, "actual balance", metrics.ActualBalanceCents, 160000)
	assertInt64Equal(t, "expected balance", metrics.ExpectedBalanceCents, 170000)
	assertFloatEqual(t, "income spent percentage", metrics.IncomeSpentPercentage, 46.6666666667)
	if !metrics.HasReceivedIncome {
		t.Fatal("expected received income flag to be true")
	}

	wantCategories := map[string]models.CategoryMetric{
		"Incomes":           {TotalAmountCents: 350000, PaidAmountCents: 300000, PendingAmountCents: 50000},
		"Fixed Expenses":    {TotalAmountCents: 150000, PaidAmountCents: 120000, PendingAmountCents: 30000},
		"Variable Expenses": {TotalAmountCents: 20000, PaidAmountCents: 20000, PendingAmountCents: 0},
		"Subscriptions":     {TotalAmountCents: 10000, PaidAmountCents: 0, PendingAmountCents: 10000},
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
		assertInt64Equal(t, category.Name+" total", category.TotalAmountCents, want.TotalAmountCents)
		assertInt64Equal(t, category.Name+" paid", category.PaidAmountCents, want.PaidAmountCents)
		assertInt64Equal(t, category.Name+" pending", category.PendingAmountCents, want.PendingAmountCents)
	}
}

func TestGetFinancialMetricsHandlesNoReceivedIncome(t *testing.T) {
	repo := newTestRepository(t)
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Pending expense", AmountCents: 5000,
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
	assertInt64Equal(t, "expected balance", metrics.ExpectedBalanceCents, -5000)
}

func TestGetFinancialMetricsAddsCentsExactly(t *testing.T) {
	repo := newTestRepository(t)
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Ten cents", AmountCents: 10,
		Date: "2026-08-10", Category: "Incomes", IsPaid: true, Active: true,
	})
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Twenty cents", AmountCents: 20,
		Date: "2026-08-11", Category: "Incomes", IsPaid: true, Active: true,
	})

	metrics, err := repo.GetFinancialMetrics("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("get exact cent metrics: %v", err)
	}
	assertInt64Equal(t, "exact received income", metrics.ReceivedIncomeCents, 30)
	assertInt64Equal(t, "exact actual balance", metrics.ActualBalanceCents, 30)
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

func TestGetSpendingReportGroupsExpensesAcrossAllDimensions(t *testing.T) {
	repo := newTestRepository(t)

	transactions := []models.Transaction{
		{
			UUID: uuid.New(), Description: "Paid rent", AmountCents: 10000,
			Date: "2026-08-01", Category: "Fixed Expenses", Subcategory: "Housing",
			PaymentMethod: "Bank Transfer", Tags: "Home, Essential", IsPaid: true, Active: true,
		},
		{
			UUID: uuid.New(), Description: "Pending groceries", AmountCents: 5000,
			Date: "2026-08-31", Category: "Variable Expenses", Subcategory: "Food",
			PaymentMethod: "Credit Card", Tags: "Food, essential, FOOD", IsPaid: false, Active: true,
		},
		{
			UUID: uuid.New(), Description: "Expense without details", AmountCents: 2500,
			Date: "2026-08-15", Category: "Variable Expenses", IsPaid: true, Active: true,
		},
		{UUID: uuid.New(), Description: "Income", AmountCents: 99999, Date: "2026-08-10", Category: "Incomes", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Archived", AmountCents: 88888, Date: "2026-08-10", Category: "Fixed Expenses", IsPaid: true, Active: false},
		{UUID: uuid.New(), Description: "Outside period", AmountCents: 77777, Date: "2026-09-01", Category: "Fixed Expenses", IsPaid: true, Active: true},
	}
	for _, transaction := range transactions {
		saveTestTransaction(t, repo, transaction)
	}

	categories, err := repo.GetCategories()
	if err != nil {
		t.Fatalf("get categories: %v", err)
	}
	for _, category := range categories {
		if category.Name == "Fixed Expenses" {
			if err := repo.SoftDeleteCategory(category.UUID); err != nil {
				t.Fatalf("inactivate historical category: %v", err)
			}
		}
	}

	report, err := repo.GetSpendingReport("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("get spending report: %v", err)
	}

	assertInt64Equal(t, "total expenses", report.TotalExpensesCents, 17500)
	assertInt64Equal(t, "paid expenses", report.PaidExpensesCents, 12500)
	assertInt64Equal(t, "pending expenses", report.PendingExpensesCents, 5000)
	if report.TransactionCount != 3 {
		t.Fatalf("expected 3 report transactions, got %d", report.TransactionCount)
	}

	assertReportGroup(t, report.ByCategory, "Fixed Expenses", 10000, 10000, 0, 1, 57.1428571429)
	assertReportGroup(t, report.ByCategory, "Variable Expenses", 7500, 2500, 5000, 2, 42.8571428571)
	assertReportGroup(t, report.BySubcategory, "Housing", 10000, 10000, 0, 1, 57.1428571429)
	assertReportGroup(t, report.BySubcategory, "Food", 5000, 0, 5000, 1, 28.5714285714)
	assertReportGroup(t, report.BySubcategory, "Unspecified", 2500, 2500, 0, 1, 14.2857142857)
	assertReportGroup(t, report.ByPaymentMethod, "Bank Transfer", 10000, 10000, 0, 1, 57.1428571429)
	assertReportGroup(t, report.ByPaymentMethod, "Credit Card", 5000, 0, 5000, 1, 28.5714285714)
	assertReportGroup(t, report.ByPaymentMethod, "Unspecified", 2500, 2500, 0, 1, 14.2857142857)
	assertReportGroup(t, report.ByTag, "Essential", 15000, 10000, 5000, 2, 85.7142857143)
	assertReportGroup(t, report.ByTag, "Home", 10000, 10000, 0, 1, 57.1428571429)
	assertReportGroup(t, report.ByTag, "Food", 5000, 0, 5000, 1, 28.5714285714)
	assertReportGroup(t, report.ByTag, "Untagged", 2500, 2500, 0, 1, 14.2857142857)

	if report.ByCategory[0].Name != "Fixed Expenses" {
		t.Fatalf("expected report groups to be sorted by total, got %#v", report.ByCategory)
	}
}

func TestGetSpendingReportReturnsEmptyGroupsAndValidatesDateRange(t *testing.T) {
	repo := newTestRepository(t)

	report, err := repo.GetSpendingReport("2026-08-01", "2026-08-31")
	if err != nil {
		t.Fatalf("get empty spending report: %v", err)
	}
	if report.ByCategory == nil || report.BySubcategory == nil || report.ByPaymentMethod == nil || report.ByTag == nil {
		t.Fatal("expected empty report dimensions to be encoded as arrays")
	}

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
			if _, err := repo.GetSpendingReport(testCase.startDate, testCase.endDate); err == nil {
				t.Fatal("expected an invalid date range error")
			}
		})
	}
}

func TestGetSpendingReportRejectsUnsafeAggregate(t *testing.T) {
	repo := newTestRepository(t)
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Maximum exact amount", AmountCents: models.MaxSafeAmountCents,
		Date: "2026-08-01", Category: "Fixed Expenses", IsPaid: true, Active: true,
	})
	saveTestTransaction(t, repo, models.Transaction{
		UUID: uuid.New(), Description: "Overflowing cent", AmountCents: 1,
		Date: "2026-08-02", Category: "Fixed Expenses", IsPaid: true, Active: true,
	})

	if _, err := repo.GetSpendingReport("2026-08-01", "2026-08-31"); err == nil {
		t.Fatal("expected an unsafe aggregate error")
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

func TestGetTransactionsSupportsArchiveStatusAndDateFilters(t *testing.T) {
	repo := newTestRepository(t)
	transactions := []models.Transaction{
		{UUID: uuid.New(), Description: "Paid August expense", AmountCents: 10000, Date: "2026-08-10", Category: "Fixed Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Pending August expense", AmountCents: 20000, Date: "2026-08-20", Category: "Fixed Expenses", IsPaid: false, Active: true},
		{UUID: uuid.New(), Description: "September expense", AmountCents: 30000, Date: "2026-09-05", Category: "Variable Expenses", IsPaid: true, Active: true},
		{UUID: uuid.New(), Description: "Archived August expense", AmountCents: 40000, Date: "2026-08-15", Category: "Fixed Expenses", IsPaid: true, Active: false},
	}
	for _, transaction := range transactions {
		saveTestTransaction(t, repo, transaction)
	}

	activeTransactions, err := repo.GetTransactions(models.TransactionFilters{})
	if err != nil {
		t.Fatalf("get active transactions: %v", err)
	}
	if len(activeTransactions) != 3 {
		t.Fatalf("expected 3 active transactions, got %d", len(activeTransactions))
	}

	allTransactions, err := repo.GetTransactions(models.TransactionFilters{IncludeArchived: true})
	if err != nil {
		t.Fatalf("get transactions including archived: %v", err)
	}
	if len(allTransactions) != 4 {
		t.Fatalf("expected 4 transactions including archived, got %d", len(allTransactions))
	}

	category := "Fixed Expenses"
	startDate := "2026-08-01"
	endDate := "2026-08-31"
	isPaid := true
	filtered, err := repo.GetTransactions(models.TransactionFilters{
		Category:        &category,
		StartDate:       &startDate,
		EndDate:         &endDate,
		IsPaid:          &isPaid,
		IncludeArchived: true,
	})
	if err != nil {
		t.Fatalf("get filtered transactions: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 paid August fixed expenses including archived, got %d", len(filtered))
	}

	amountCents := int64(10000)
	amountFiltered, err := repo.GetTransactions(models.TransactionFilters{
		AmountCents:     &amountCents,
		IncludeArchived: true,
	})
	if err != nil {
		t.Fatalf("get transactions filtered by exact cents: %v", err)
	}
	if len(amountFiltered) != 1 || amountFiltered[0].Description != "Paid August expense" {
		t.Fatalf("unexpected exact cent filter results: %#v", amountFiltered)
	}
}

func TestUpdateArchiveAndRestoreTransaction(t *testing.T) {
	repo := newTestRepository(t)
	transactionID := uuid.New()
	saveTestTransaction(t, repo, models.Transaction{
		UUID: transactionID, Description: "Original expense", AmountCents: 5000,
		Date: "2026-08-10", Category: "Fixed Expenses", Active: true,
	})
	if err := repo.MarkAsNotified(transactionID.String(), "2026-08-10"); err != nil {
		t.Fatalf("mark original transaction as notified: %v", err)
	}

	updated := models.Transaction{
		UUID:          transactionID,
		Description:   "Updated expense",
		AmountCents:   7525,
		Date:          "2026-08-12",
		Category:      "Variable Expenses",
		Subcategory:   "Food",
		PaymentMethod: "Credit Card",
		Installments:  "1 of 3",
		Tags:          "#test, #updated",
		IsPaid:        true,
		Active:        true,
	}
	if err := repo.UpdateTransaction(updated); err != nil {
		t.Fatalf("update transaction: %v", err)
	}

	stored, err := repo.GetTransactionByID(transactionID.String())
	if err != nil {
		t.Fatalf("get updated transaction: %v", err)
	}
	if stored.Description != updated.Description ||
		stored.AmountCents != updated.AmountCents ||
		stored.Date != updated.Date ||
		stored.Category != updated.Category ||
		stored.Subcategory != updated.Subcategory ||
		stored.PaymentMethod != updated.PaymentMethod ||
		stored.Installments != updated.Installments ||
		stored.Tags != updated.Tags ||
		stored.IsPaid != updated.IsPaid {
		t.Fatalf("stored transaction does not match update: %#v", stored)
	}

	var notifiedAt string
	if err := repo.db.QueryRow(
		"SELECT notified_at FROM transactions WHERE uuid = ?;",
		transactionID.String(),
	).Scan(&notifiedAt); err != nil {
		t.Fatalf("read updated reminder state: %v", err)
	}
	if notifiedAt != "" {
		t.Fatalf("expected update to reset reminder state, got %q", notifiedAt)
	}

	if err := repo.SoftDeleteTransaction(transactionID.String()); err != nil {
		t.Fatalf("archive transaction: %v", err)
	}
	if err := repo.UpdateTransaction(updated); err == nil {
		t.Fatal("expected archived transaction update to fail")
	}
	if err := repo.RestoreTransaction(transactionID.String()); err != nil {
		t.Fatalf("restore transaction: %v", err)
	}
	if _, err := repo.GetTransactionByID(transactionID.String()); err != nil {
		t.Fatalf("get restored transaction: %v", err)
	}
	if err := repo.RestoreTransaction(transactionID.String()); err == nil {
		t.Fatal("expected restoring an active transaction to fail")
	}
}

func TestSaveTransactionRejectsInexactRange(t *testing.T) {
	repo := newTestRepository(t)
	testCases := []struct {
		name        string
		amountCents int64
	}{
		{name: "zero", amountCents: 0},
		{name: "negative", amountCents: -1},
		{name: "above JavaScript safe integer", amountCents: models.MaxSafeAmountCents + 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := repo.SaveTransaction(models.Transaction{
				UUID: uuid.New(), Description: testCase.name, AmountCents: testCase.amountCents,
				Date: "2026-08-01", Category: "Fixed Expenses", Active: true,
			})
			if err == nil {
				t.Fatal("expected invalid cent amount to be rejected")
			}
		})
	}
}

func assertFloatEqual(t *testing.T, label string, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Errorf("%s: expected %.10f, got %.10f", label, want, got)
	}
}

func assertInt64Equal(t *testing.T, label string, got int64, want int64) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected %d, got %d", label, want, got)
	}
}

func assertReportGroup(
	t *testing.T,
	groups []models.ReportGroup,
	name string,
	totalCents int64,
	paidCents int64,
	pendingCents int64,
	transactionCount int,
	percentage float64,
) {
	t.Helper()
	for _, group := range groups {
		if strings.EqualFold(group.Name, name) {
			assertInt64Equal(t, name+" total", group.TotalAmountCents, totalCents)
			assertInt64Equal(t, name+" paid", group.PaidAmountCents, paidCents)
			assertInt64Equal(t, name+" pending", group.PendingAmountCents, pendingCents)
			if group.TransactionCount != transactionCount {
				t.Errorf("%s: expected %d transactions, got %d", name, transactionCount, group.TransactionCount)
			}
			assertFloatEqual(t, name+" percentage", group.PercentageOfExpenses, percentage)
			return
		}
	}
	t.Errorf("expected report group %q in %#v", name, groups)
}
