//go:build windows

package notifier

import (
	"strings"

	"gopkg.in/toast.v1"
)

type windowsNotificationSender struct{}

func newNotificationSender() notificationSender {
	return windowsNotificationSender{}
}

func (windowsNotificationSender) Send(title string, message string) error {
	notification := toast.Notification{
		AppID:   "Prisma Finances",
		Title:   escapeWindowsToastText(title),
		Message: escapeWindowsToastText(message),
	}

	return notification.Push()
}

// escapeWindowsToastText prevents PowerShell from expanding notification content
// inside the expandable here-string used by the toast package.
func escapeWindowsToastText(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "`", "``")
	value = strings.ReplaceAll(value, "$", "`$")
	value = strings.ReplaceAll(value, "]]>", "]]]]><![CDATA[>")
	return value
}
