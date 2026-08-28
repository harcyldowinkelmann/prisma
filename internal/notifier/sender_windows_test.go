//go:build windows

package notifier

import "testing"

func TestEscapeWindowsToastText(t *testing.T) {
	input := "Expense $125.00 with `code`\r\nand ]]> content"
	want := "Expense `$125.00 with ``code``  and ]]]]><![CDATA[> content"

	if got := escapeWindowsToastText(input); got != want {
		t.Fatalf("unexpected escaped text:\nwant: %q\n got: %q", want, got)
	}
}
