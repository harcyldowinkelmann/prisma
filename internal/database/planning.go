package database

import (
	"database/sql"
	"fmt"
	"prisma/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SaveInstallmentTransactions splits a total into exact monthly installments atomically.
func (r *Repository) SaveInstallmentTransactions(transaction models.Transaction, installmentCount int) error {
	if installmentCount < 1 || installmentCount > 120 {
		return fmt.Errorf("installment count must be between 1 and 120")
	}
	if err := validateAmountCents(transaction.AmountCents); err != nil {
		return err
	}
	startDate, err := time.Parse("2006-01-02", transaction.Date)
	if err != nil {
		return fmt.Errorf("invalid installment start date: %w", err)
	}
	if err := r.validateActiveCategory(transaction.Category, 0); err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting installment plan: %w", err)
	}
	defer tx.Rollback()

	baseAmount := transaction.AmountCents / int64(installmentCount)
	remainder := transaction.AmountCents % int64(installmentCount)
	if baseAmount == 0 {
		return fmt.Errorf("the total amount must contain at least one cent per installment")
	}
	groupID := ""
	if installmentCount > 1 {
		groupID = uuid.New().String()
	}

	for index := 0; index < installmentCount; index++ {
		amountCents := baseAmount
		if int64(index) < remainder {
			amountCents++
		}
		installments := ""
		if installmentCount > 1 {
			installments = fmt.Sprintf("%d/%d", index+1, installmentCount)
		}
		installmentDate := addMonthsClamped(startDate, index).Format("2006-01-02")
		if _, err := tx.Exec(`
			INSERT INTO transactions (
				uuid, description, amount_cents, date, category, subcategory,
				payment_method, installments, tags, is_paid, reconciled,
				installment_group, active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, 1);
		`, uuid.New(), transaction.Description, amountCents, installmentDate,
			transaction.Category, transaction.Subcategory, transaction.PaymentMethod,
			installments, transaction.Tags, transaction.IsPaid, groupID); err != nil {
			return fmt.Errorf("error saving installment %d of %d: %w", index+1, installmentCount, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing installment plan: %w", err)
	}
	return nil
}

// SaveRecurringSchedule validates and stores a recurring transaction rule.
func (r *Repository) SaveRecurringSchedule(schedule models.RecurringSchedule) error {
	if strings.TrimSpace(schedule.Description) == "" {
		return fmt.Errorf("recurring description is required")
	}
	if err := validateAmountCents(schedule.AmountCents); err != nil {
		return err
	}
	start, err := time.Parse("2006-01-02", schedule.StartDate)
	if err != nil {
		return fmt.Errorf("invalid recurring start date: %w", err)
	}
	if schedule.EndDate != "" {
		end, err := time.Parse("2006-01-02", schedule.EndDate)
		if err != nil {
			return fmt.Errorf("invalid recurring end date: %w", err)
		}
		if end.Before(start) {
			return fmt.Errorf("recurring end date must not be before the start date")
		}
	}
	if schedule.Frequency != "weekly" && schedule.Frequency != "monthly" && schedule.Frequency != "yearly" {
		return fmt.Errorf("unsupported recurring frequency: %s", schedule.Frequency)
	}
	if err := r.validateActiveCategory(schedule.Category, 0); err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO recurring_schedules (
			uuid, description, amount_cents, start_date, end_date, frequency,
			category, subcategory, payment_method, tags, is_paid, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1);
	`, schedule.UUID, strings.TrimSpace(schedule.Description), schedule.AmountCents,
		schedule.StartDate, schedule.EndDate, schedule.Frequency, schedule.Category,
		strings.TrimSpace(schedule.Subcategory), strings.TrimSpace(schedule.PaymentMethod),
		strings.TrimSpace(schedule.Tags), schedule.IsPaid)
	if err != nil {
		return fmt.Errorf("error saving recurring schedule: %w", err)
	}
	return nil
}

// GetRecurringSchedules returns active rules ordered by their first occurrence.
func (r *Repository) GetRecurringSchedules() ([]models.RecurringSchedule, error) {
	rows, err := r.db.Query(`
		SELECT uuid, description, amount_cents, start_date, end_date, frequency,
		       category, subcategory, payment_method, tags, is_paid, active
		FROM recurring_schedules
		WHERE active = 1
		ORDER BY start_date, description;
	`)
	if err != nil {
		return nil, fmt.Errorf("error fetching recurring schedules: %w", err)
	}
	defer rows.Close()

	schedules := []models.RecurringSchedule{}
	for rows.Next() {
		var schedule models.RecurringSchedule
		if err := rows.Scan(
			&schedule.UUID, &schedule.Description, &schedule.AmountCents,
			&schedule.StartDate, &schedule.EndDate, &schedule.Frequency,
			&schedule.Category, &schedule.Subcategory, &schedule.PaymentMethod,
			&schedule.Tags, &schedule.IsPaid, &schedule.Active,
		); err != nil {
			return nil, fmt.Errorf("error scanning recurring schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recurring schedules: %w", err)
	}
	return schedules, nil
}

// SoftDeleteRecurringSchedule stops future generation without changing past occurrences.
func (r *Repository) SoftDeleteRecurringSchedule(scheduleUUID string) error {
	result, err := r.db.Exec("UPDATE recurring_schedules SET active = 0 WHERE uuid = ? AND active = 1;", scheduleUUID)
	if err != nil {
		return fmt.Errorf("error stopping recurring schedule: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 1 {
		return fmt.Errorf("no active recurring schedule found with UUID: %s", scheduleUUID)
	}
	return nil
}

// GenerateRecurringTransactions creates missing occurrences through an inclusive date.
func (r *Repository) GenerateRecurringTransactions(throughDate string) (int, error) {
	through, err := time.Parse("2006-01-02", throughDate)
	if err != nil {
		return 0, fmt.Errorf("invalid recurring generation date: %w", err)
	}
	schedules, err := r.GetRecurringSchedules()
	if err != nil {
		return 0, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error starting recurring generation: %w", err)
	}
	defer tx.Rollback()
	generatedCount := 0
	for _, schedule := range schedules {
		start, _ := time.Parse("2006-01-02", schedule.StartDate)
		var end time.Time
		if schedule.EndDate != "" {
			end, _ = time.Parse("2006-01-02", schedule.EndDate)
		}
		for occurrenceIndex := 0; occurrenceIndex < 10000; occurrenceIndex++ {
			occurrence := recurringOccurrence(start, schedule.Frequency, occurrenceIndex)
			if occurrence.After(through) || (!end.IsZero() && occurrence.After(end)) {
				break
			}
			occurrenceDate := occurrence.Format("2006-01-02")
			frequencyTitle := map[string]string{
				"weekly": "Weekly", "monthly": "Monthly", "yearly": "Yearly",
			}[schedule.Frequency]
			if frequencyTitle == "" {
				return 0, fmt.Errorf("recurring schedule %s has an unsupported frequency", schedule.UUID)
			}
			installmentLabel := "Recurring: " + frequencyTitle
			result, err := tx.Exec(`
				INSERT OR IGNORE INTO transactions (
					uuid, description, amount_cents, date, category, subcategory,
					payment_method, installments, tags, is_paid, reconciled,
					recurrence_id, occurrence_date, active
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, 1);
			`, uuid.New(), schedule.Description, schedule.AmountCents, occurrenceDate,
				schedule.Category, schedule.Subcategory, schedule.PaymentMethod,
				installmentLabel, schedule.Tags, schedule.IsPaid, schedule.UUID, occurrenceDate)
			if err != nil {
				return 0, fmt.Errorf("error generating recurring occurrence: %w", err)
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return 0, fmt.Errorf("error verifying recurring occurrence: %w", err)
			}
			generatedCount += int(rowsAffected)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error committing recurring generation: %w", err)
	}
	return generatedCount, nil
}

// SaveBudget creates or replaces a monthly category limit.
func (r *Repository) SaveBudget(month string, category string, limitCents int64) error {
	if _, err := time.Parse("2006-01", month); err != nil {
		return fmt.Errorf("invalid budget month: %w", err)
	}
	if err := validateAmountCents(limitCents); err != nil {
		return fmt.Errorf("invalid budget limit: %w", err)
	}
	if err := r.validateActiveCategory(category, -1); err != nil {
		return err
	}
	_, err := r.db.Exec(`
		INSERT INTO budgets (uuid, month, category, limit_cents)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(month, category) DO UPDATE SET limit_cents = excluded.limit_cents;
	`, uuid.New(), month, category, limitCents)
	if err != nil {
		return fmt.Errorf("error saving budget: %w", err)
	}
	return nil
}

// DeleteBudget removes one monthly category limit.
func (r *Repository) DeleteBudget(month string, category string) error {
	result, err := r.db.Exec("DELETE FROM budgets WHERE month = ? AND category = ?;", month, category)
	if err != nil {
		return fmt.Errorf("error deleting budget: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 1 {
		return fmt.Errorf("no budget found for %s in %s", category, month)
	}
	return nil
}

// GetBudgetSummaries calculates category spending for one calendar month.
func (r *Repository) GetBudgetSummaries(month string) ([]models.BudgetSummary, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, fmt.Errorf("invalid budget month: %w", err)
	}
	startDate := start.Format("2006-01-02")
	endDate := addMonthsClamped(start, 1).AddDate(0, 0, -1).Format("2006-01-02")
	rows, err := r.db.Query(`
		SELECT b.uuid, b.month, b.category, b.limit_cents,
		       COALESCE(SUM(t.amount_cents), 0)
		FROM budgets b
		LEFT JOIN transactions t
		       ON t.category = b.category
		      AND t.active = 1
		      AND t.date BETWEEN ? AND ?
		WHERE b.month = ?
		GROUP BY b.uuid, b.month, b.category, b.limit_cents
		ORDER BY b.category;
	`, startDate, endDate, month)
	if err != nil {
		return nil, fmt.Errorf("error calculating budgets: %w", err)
	}
	defer rows.Close()

	summaries := []models.BudgetSummary{}
	for rows.Next() {
		var summary models.BudgetSummary
		if err := rows.Scan(&summary.UUID, &summary.Month, &summary.Category, &summary.LimitCents, &summary.SpentCents); err != nil {
			return nil, fmt.Errorf("error scanning budget: %w", err)
		}
		if summary.SpentCents > models.MaxSafeAmountCents {
			return nil, fmt.Errorf("budget spending exceeds the maximum exact supported value")
		}
		summary.RemainingCents = summary.LimitCents - summary.SpentCents
		summary.PercentageUsed = float64(summary.SpentCents) / float64(summary.LimitCents) * 100
		summary.OverBudget = summary.SpentCents > summary.LimitCents
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating budgets: %w", err)
	}
	return summaries, nil
}

func (r *Repository) validateActiveCategory(category string, expectedType int) error {
	var categoryType int
	if err := r.db.QueryRow(
		"SELECT type FROM categories WHERE name = ? AND active = 1;",
		strings.TrimSpace(category),
	).Scan(&categoryType); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("choose an active category")
		}
		return fmt.Errorf("error validating category: %w", err)
	}
	if expectedType != 0 && categoryType != expectedType {
		return fmt.Errorf("the selected category has the wrong transaction type")
	}
	return nil
}

func recurringOccurrence(start time.Time, frequency string, index int) time.Time {
	switch frequency {
	case "weekly":
		return start.AddDate(0, 0, index*7)
	case "yearly":
		return addMonthsClamped(start, index*12)
	default:
		return addMonthsClamped(start, index)
	}
}

func addMonthsClamped(date time.Time, months int) time.Time {
	firstOfTarget := time.Date(date.Year(), date.Month()+time.Month(months), 1, 0, 0, 0, 0, date.Location())
	lastDay := time.Date(firstOfTarget.Year(), firstOfTarget.Month()+1, 0, 0, 0, 0, 0, date.Location()).Day()
	day := date.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), day, 0, 0, 0, 0, date.Location())
}
