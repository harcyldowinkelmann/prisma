package database

import (
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"prisma/internal/models"
)

func TestBackupRoundTripPreservesAllData(t *testing.T) {
	repo := newTestRepository(t)
	createdAt := time.Date(2026, time.August, 31, 14, 30, 0, 0, time.UTC)
	want := completeTestBackup(createdAt)

	if err := repo.RestoreBackup(want); err != nil {
		t.Fatalf("restore complete backup: %v", err)
	}
	got, err := repo.BuildBackup(createdAt)
	if err != nil {
		t.Fatalf("build restored backup: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("backup round trip changed data\nwant: %#v\n got: %#v", want, got)
	}
}

func TestRestoreBackupRejectsInvalidDataWithoutChanges(t *testing.T) {
	repo := newTestRepository(t)
	createdAt := time.Date(2026, time.August, 31, 15, 0, 0, 0, time.UTC)
	baseline := completeTestBackup(createdAt)
	if err := repo.RestoreBackup(baseline); err != nil {
		t.Fatalf("restore baseline backup: %v", err)
	}

	testCases := []struct {
		name   string
		change func(*models.BackupData)
	}{
		{
			name: "unsupported format",
			change: func(backup *models.BackupData) {
				backup.FormatVersion++
			},
		},
		{
			name: "zero transaction amount",
			change: func(backup *models.BackupData) {
				backup.Transactions[0].AmountCents = 0
			},
		},
		{
			name: "unknown transaction category",
			change: func(backup *models.BackupData) {
				backup.Transactions[0].Category = "Unknown"
			},
		},
		{
			name: "duplicate statement fingerprint",
			change: func(backup *models.BackupData) {
				backup.Transactions[1].ImportFingerprint = backup.Transactions[0].ImportFingerprint
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			invalid := cloneTestBackup(t, baseline)
			testCase.change(&invalid)
			if err := repo.RestoreBackup(invalid); err == nil {
				t.Fatal("expected invalid backup to be rejected")
			}
			got, err := repo.BuildBackup(createdAt)
			if err != nil {
				t.Fatalf("build backup after rejection: %v", err)
			}
			if !reflect.DeepEqual(got, baseline) {
				t.Fatal("invalid restore changed the existing data")
			}
		})
	}
}

func TestRestoreBackupRollsBackDatabaseFailure(t *testing.T) {
	repo := newTestRepository(t)
	createdAt := time.Date(2026, time.August, 31, 16, 0, 0, 0, time.UTC)
	baseline := completeTestBackup(createdAt)
	if err := repo.RestoreBackup(baseline); err != nil {
		t.Fatalf("restore baseline backup: %v", err)
	}

	if _, err := repo.db.Exec(`
		CREATE TRIGGER fail_backup_category_insert
		BEFORE INSERT ON categories
		BEGIN
			SELECT RAISE(ABORT, 'forced restore failure');
		END;
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	if err := repo.RestoreBackup(baseline); err == nil {
		t.Fatal("expected forced database failure")
	}
	if _, err := repo.db.Exec("DROP TRIGGER fail_backup_category_insert;"); err != nil {
		t.Fatalf("drop failure trigger: %v", err)
	}

	got, err := repo.BuildBackup(createdAt)
	if err != nil {
		t.Fatalf("build backup after rollback: %v", err)
	}
	if !reflect.DeepEqual(got, baseline) {
		t.Fatal("failed restore was not rolled back atomically")
	}
}

func TestBuildTransactionsCSVUsesExactAmountsAndStatuses(t *testing.T) {
	repo := newTestRepository(t)
	backup := completeTestBackup(time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC))
	if err := repo.RestoreBackup(backup); err != nil {
		t.Fatalf("restore CSV fixture: %v", err)
	}

	content, err := repo.BuildTransactionsCSV()
	if err != nil {
		t.Fatalf("build transaction CSV: %v", err)
	}
	if !strings.HasPrefix(string(content), "\ufeff") {
		t.Fatal("expected UTF-8 BOM for spreadsheet compatibility")
	}
	records, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(content), "\ufeff"))).ReadAll()
	if err != nil {
		t.Fatalf("parse generated CSV: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected header and two transactions, got %d rows", len(records))
	}
	wantHeaders := []string{
		"ID", "Description", "Amount", "Date", "Category", "Subcategory",
		"Payment Method", "Installments", "Tags", "Payment Status",
		"Reconciliation Status", "Record Status",
	}
	if !reflect.DeepEqual(records[0], wantHeaders) {
		t.Fatalf("unexpected CSV headers: %#v", records[0])
	}

	rowsByDescription := make(map[string][]string)
	for _, row := range records[1:] {
		rowsByDescription[row[1]] = row
	}
	income := rowsByDescription[`Salary, "August"`]
	if len(income) == 0 {
		t.Fatal("quoted income description was not preserved")
	}
	if income[2] != "1234.56" || income[9] != "Paid" || income[10] != "Reconciled" || income[11] != "Active" {
		t.Fatalf("unexpected income CSV values: %#v", income)
	}
	expense := rowsByDescription["Archived rent"]
	if expense[2] != "999.01" || expense[9] != "Pending" || expense[10] != "Unreconciled" || expense[11] != "Archived" {
		t.Fatalf("unexpected expense CSV values: %#v", expense)
	}
}

func completeTestBackup(createdAt time.Time) models.BackupData {
	incomeCategoryID := "11111111-1111-4111-8111-111111111111"
	expenseCategoryID := "22222222-2222-4222-8222-222222222222"
	recurrenceID := "33333333-3333-4333-8333-333333333333"
	return models.BackupData{
		FormatVersion: models.BackupFormatVersion,
		CreatedAt:     createdAt.UTC().Format(time.RFC3339),
		Transactions: []models.BackupTransaction{
			{
				UUID: "44444444-4444-4444-8444-444444444444", Description: `Salary, "August"`,
				AmountCents: 123456, Date: "2026-08-10", Category: "Income",
				Subcategory: "Salary", PaymentMethod: "Bank Transfer", Installments: "",
				Tags: "Work", IsPaid: true, Reconciled: true,
				ImportFingerprint: "statement-fingerprint-1", NotifiedAt: "2026-08-10", Active: true,
			},
			{
				UUID: "55555555-5555-4555-8555-555555555555", Description: "Archived rent",
				AmountCents: 99901, Date: "2026-08-15", Category: "Fixed Expenses",
				Subcategory: "Housing", PaymentMethod: "Credit Card", Installments: "Recurring: Monthly",
				Tags: "Home", IsPaid: false, Reconciled: false,
				ImportFingerprint: "statement-fingerprint-2", RecurrenceID: recurrenceID,
				OccurrenceDate: "2026-08-15", InstallmentGroup: "66666666-6666-4666-8666-666666666666",
				Active: false,
			},
		},
		Categories: []models.Category{
			{UUID: incomeCategoryID, Name: "Income", Type: 1, Active: true},
			{UUID: expenseCategoryID, Name: "Fixed Expenses", Type: -1, Active: false},
		},
		Subcategories: []models.SettingItem{
			{UUID: "77777777-7777-4777-8777-777777777777", Name: "Housing", Active: false},
			{UUID: "88888888-8888-4888-8888-888888888888", Name: "Salary", Active: true},
		},
		PaymentMethods: []models.SettingItem{
			{UUID: "99999999-9999-4999-8999-999999999999", Name: "Bank Transfer", Active: true},
			{UUID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Name: "Credit Card", Active: false},
		},
		Tags: []models.SettingItem{
			{UUID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Name: "Home", Active: false},
			{UUID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", Name: "Work", Active: true},
		},
		RecurringSchedules: []models.RecurringSchedule{
			{
				UUID: recurrenceID, Description: "Archived rent", AmountCents: 99901,
				StartDate: "2026-08-15", EndDate: "2027-08-15", Frequency: "monthly",
				Category: "Fixed Expenses", Subcategory: "Housing", PaymentMethod: "Credit Card",
				Tags: "Home", IsPaid: false, Active: false,
			},
		},
		Budgets: []models.BackupBudget{
			{UUID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Month: "2026-08", Category: "Fixed Expenses", LimitCents: 150000},
		},
		AppSettings: []models.BackupSetting{
			{Key: currencyCodeSettingKey, Value: "BRL"},
			{Key: notificationsEnabledSettingKey, Value: "false"},
		},
	}
}

func cloneTestBackup(t *testing.T, source models.BackupData) models.BackupData {
	t.Helper()
	content, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("encode test backup: %v", err)
	}
	var clone models.BackupData
	if err := json.Unmarshal(content, &clone); err != nil {
		t.Fatalf("decode test backup: %v", err)
	}
	return clone
}
