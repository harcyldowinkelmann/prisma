//go:build !windows

package notifier

func newNotificationSender() notificationSender {
	return nil
}
