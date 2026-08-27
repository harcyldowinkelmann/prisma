package notifier

import (
	"fmt"
	"log"
	"prisma/internal/database"
	"time"

	"gopkg.in/toast.v1"
)

// StartBackgroundNotifier starts a background goroutine that checks for unpaid bills
func StartBackgroundNotifier(repo *database.Repository) {
	// Execute the check immediately on startup
	go func() {
		log.Println("Starting background notification service...")
		checkAndNotify(repo)

		// Then run it periodically (e.g. every 1 hour)
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			checkAndNotify(repo)
		}
	}()
}

func checkAndNotify(repo *database.Repository) {
	// Assuming date format is YYYY-MM-DD
	todayDate := time.Now().Format("2006-01-02")
	
	pending, err := repo.GetPendingNotifications(todayDate)
	if err != nil {
		log.Printf("Notifier error fetching pending transactions: %v\n", err)
		return
	}

	for _, p := range pending {
		// Prepare the toast notification
		notification := toast.Notification{
			AppID:   "Prisma Finances",
			Title:   "Lembrete de Pagamento",
			Message: fmt.Sprintf("Você tem uma conta pendente: %s no valor de R$ %.2f", p.Description, p.Amount),
		}

		// Try to push the notification
		err := notification.Push()
		if err != nil {
			log.Printf("Failed to push Windows notification: %v\n", err)
			continue
		}

		// Mark as notified in the database so we don't spam the user today
		err = repo.MarkAsNotified(p.UUID.String(), todayDate)
		if err != nil {
			log.Printf("Failed to mark as notified: %v\n", err)
		}
	}
}
