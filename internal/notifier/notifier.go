package notifier

import (
	"context"
	"fmt"
	"log"
	"prisma/internal/database"
	"time"
)

const notificationCheckInterval = time.Hour

type notificationSender interface {
	Send(title string, message string) error
}

// StartBackgroundNotifier checks for unpaid expenses until the application context is canceled.
// The returned channel is closed after the background worker has stopped.
func StartBackgroundNotifier(ctx context.Context, repo *database.Repository) <-chan struct{} {
	done := make(chan struct{})
	sender := newNotificationSender()
	if sender == nil {
		log.Println("Desktop notifications are not supported on this operating system.")
		close(done)
		return done
	}

	go func() {
		defer close(done)
		log.Println("Starting background notification service...")
		select {
		case <-ctx.Done():
			log.Println("Background notification service stopped.")
			return
		default:
		}

		checkAndNotify(repo, sender)

		ticker := time.NewTicker(notificationCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Background notification service stopped.")
				return
			case <-ticker.C:
				checkAndNotify(repo, sender)
			}
		}
	}()

	return done
}

func checkAndNotify(repo *database.Repository, sender notificationSender) {
	enabled, err := repo.GetNotificationsEnabled()
	if err != nil {
		log.Printf("Failed to read notification settings: %v", err)
		return
	}
	if !enabled {
		return
	}

	todayDate := time.Now().Format("2006-01-02")
	pending, err := repo.GetPendingNotifications(todayDate)
	if err != nil {
		log.Printf("Failed to fetch pending expenses: %v", err)
		return
	}

	for _, transaction := range pending {
		message := fmt.Sprintf("You have an unpaid expense: %s for $%.2f", transaction.Description, transaction.Amount)
		if err := sender.Send("Payment Reminder", message); err != nil {
			log.Printf("Failed to send payment reminder: %v", err)
			continue
		}

		if err := repo.MarkAsNotified(transaction.UUID.String(), todayDate); err != nil {
			log.Printf("Failed to mark transaction as notified: %v", err)
		}
	}
}
