package notifier

import (
	"context"
	"errors"
	"prisma/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeNotificationRepository struct {
	enabled      bool
	settingsErr  error
	pending      []models.Transaction
	pendingErr   error
	pendingCalls int
	marked       []string
	currencyCode string
	currencyErr  error
}

func (r *fakeNotificationRepository) GetNotificationsEnabled() (bool, error) {
	return r.enabled, r.settingsErr
}

func (r *fakeNotificationRepository) GetCurrencyCode() (string, error) {
	if r.currencyCode == "" {
		return "USD", r.currencyErr
	}
	return r.currencyCode, r.currencyErr
}

func (r *fakeNotificationRepository) GetPendingNotifications(string) ([]models.Transaction, error) {
	r.pendingCalls++
	return r.pending, r.pendingErr
}

func (r *fakeNotificationRepository) MarkAsNotified(id string, date string) error {
	r.marked = append(r.marked, id+"@"+date)
	return nil
}

type sentNotification struct {
	title   string
	message string
}

type fakeNotificationSender struct {
	sent      []sentNotification
	failCalls map[int]bool
	calls     int
}

func (s *fakeNotificationSender) Send(title string, message string) error {
	s.calls++
	if s.failCalls[s.calls] {
		return errors.New("notification delivery failed")
	}
	s.sent = append(s.sent, sentNotification{title: title, message: message})
	return nil
}

func TestCheckAndNotifySkipsWorkWhenDisabled(t *testing.T) {
	repo := &fakeNotificationRepository{enabled: false}
	sender := &fakeNotificationSender{}

	checkAndNotify(repo, sender, "2026-08-26")

	if repo.pendingCalls != 0 {
		t.Fatalf("expected no pending query, got %d", repo.pendingCalls)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("expected no notifications, got %#v", sender.sent)
	}
}

func TestCheckAndNotifySendsEnglishMessageAndMarksTransaction(t *testing.T) {
	id := uuid.New()
	repo := &fakeNotificationRepository{
		enabled: true,
		pending: []models.Transaction{{
			UUID: id, Description: "Internet bill", AmountCents: 12545,
		}},
	}
	sender := &fakeNotificationSender{}

	checkAndNotify(repo, sender, "2026-08-26")

	if len(sender.sent) != 1 {
		t.Fatalf("expected one notification, got %#v", sender.sent)
	}
	if sender.sent[0].title != "Payment Reminder" {
		t.Errorf("unexpected title %q", sender.sent[0].title)
	}
	if sender.sent[0].message != "You have an unpaid expense: Internet bill for $125.45" {
		t.Errorf("unexpected message %q", sender.sent[0].message)
	}
	if len(repo.marked) != 1 || repo.marked[0] != id.String()+"@2026-08-26" {
		t.Fatalf("unexpected marked transactions: %#v", repo.marked)
	}
}

func TestCheckAndNotifyDoesNotMarkFailedDeliveryAndContinues(t *testing.T) {
	failedID := uuid.New()
	successID := uuid.New()
	repo := &fakeNotificationRepository{
		enabled: true,
		pending: []models.Transaction{
			{UUID: failedID, Description: "First expense", AmountCents: 1000},
			{UUID: successID, Description: "Second expense", AmountCents: 2000},
		},
	}
	sender := &fakeNotificationSender{failCalls: map[int]bool{1: true}}

	checkAndNotify(repo, sender, "2026-08-26")

	if sender.calls != 2 {
		t.Fatalf("expected two delivery attempts, got %d", sender.calls)
	}
	if len(repo.marked) != 1 || repo.marked[0] != successID.String()+"@2026-08-26" {
		t.Fatalf("expected only the successful delivery to be marked, got %#v", repo.marked)
	}
}

func TestCheckAndNotifyUsesConfiguredCurrency(t *testing.T) {
	repo := &fakeNotificationRepository{
		enabled:      true,
		currencyCode: "BRL",
		pending: []models.Transaction{{
			UUID: uuid.New(), Description: "Electricity bill", AmountCents: 9876,
		}},
	}
	sender := &fakeNotificationSender{}

	checkAndNotify(repo, sender, "2026-08-26")

	if len(sender.sent) != 1 {
		t.Fatalf("expected one notification, got %#v", sender.sent)
	}
	if sender.sent[0].message != "You have an unpaid expense: Electricity bill for R$98.76" {
		t.Errorf("unexpected message %q", sender.sent[0].message)
	}
}

func TestRunBackgroundNotifierStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := &fakeNotificationRepository{enabled: true}
	sender := &fakeNotificationSender{}
	done := runBackgroundNotifier(ctx, repo, sender, time.Hour)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("background notifier did not stop after cancellation")
	}
	if repo.pendingCalls != 0 {
		t.Fatalf("expected no database work after cancellation, got %d queries", repo.pendingCalls)
	}
}
