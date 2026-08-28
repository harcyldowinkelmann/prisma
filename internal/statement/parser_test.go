package statement

import (
	"prisma/internal/models"
	"testing"
)

func TestInspectDetectsSemicolonColumnsAndEuropeanDate(t *testing.T) {
	content := "Date;Description;Debit;Credit\n27/08/2026;Coffee;12,34;\n"
	inspection, err := Inspect(content, "auto", true)
	if err != nil {
		t.Fatalf("inspect statement: %v", err)
	}
	if inspection.Delimiter != "semicolon" {
		t.Fatalf("expected semicolon delimiter, got %q", inspection.Delimiter)
	}
	if inspection.DetectedDateColumn != 0 || inspection.DetectedDescriptionColumn != 1 ||
		inspection.DetectedDebitColumn != 2 || inspection.DetectedCreditColumn != 3 {
		t.Fatalf("unexpected detected columns: %#v", inspection)
	}
	if inspection.DetectedDateFormat != "dd/mm/yyyy" {
		t.Fatalf("expected DD/MM/YYYY detection, got %q", inspection.DetectedDateFormat)
	}
}

func TestParseSignedAmountsUsesExactCentsAndStableOccurrences(t *testing.T) {
	content := "Date;Description;Amount\n27/08/2026;Coffee;-12,34\n27/08/2026;Coffee;-12,34\n28/08/2026;Salary;1.234,56\n"
	options := models.StatementParseOptions{
		Delimiter: "semicolon", HasHeader: true, DateColumn: 0, DescriptionColumn: 1,
		AmountMode: "signed", AmountColumn: 2, DateFormat: "dd/mm/yyyy", DecimalSeparator: "auto",
	}
	preview, err := Parse(content, options)
	if err != nil {
		t.Fatalf("parse statement: %v", err)
	}
	if len(preview.Rows) != 3 || len(preview.Errors) != 0 {
		t.Fatalf("unexpected statement preview: %#v", preview)
	}
	if preview.Rows[0].Date != "2026-08-27" || preview.Rows[0].AmountCents != 1234 || preview.Rows[0].Type != -1 {
		t.Fatalf("unexpected expense row: %#v", preview.Rows[0])
	}
	if preview.Rows[2].AmountCents != 123456 || preview.Rows[2].Type != 1 {
		t.Fatalf("unexpected income row: %#v", preview.Rows[2])
	}
	if preview.Rows[0].Fingerprint == preview.Rows[1].Fingerprint || preview.Rows[1].Occurrence != 2 {
		t.Fatal("expected repeated legitimate rows to receive distinct occurrence fingerprints")
	}

	repeatedPreview, err := Parse(content, options)
	if err != nil {
		t.Fatalf("repeat statement parse: %v", err)
	}
	for index := range preview.Rows {
		if preview.Rows[index].Fingerprint != repeatedPreview.Rows[index].Fingerprint {
			t.Fatalf("fingerprint changed at row %d", index+1)
		}
	}
}

func TestParseDebitAndCreditColumnsReportsInvalidRows(t *testing.T) {
	content := "Date,Description,Debit,Credit\n08/27/2026,Rent,100.00,\n08/28/2026,Refund,,25.50\ninvalid,Broken,1.00,2.00\n"
	options := models.StatementParseOptions{
		Delimiter: "comma", HasHeader: true, DateColumn: 0, DescriptionColumn: 1,
		AmountMode: "debit_credit", DebitColumn: 2, CreditColumn: 3,
		DateFormat: "mm/dd/yyyy", DecimalSeparator: "dot", AmountColumn: -1,
	}
	preview, err := Parse(content, options)
	if err != nil {
		t.Fatalf("parse debit and credit statement: %v", err)
	}
	if len(preview.Rows) != 2 || len(preview.Errors) != 1 {
		t.Fatalf("unexpected statement preview: %#v", preview)
	}
	if preview.Rows[0].Type != -1 || preview.Rows[0].AmountCents != 10000 {
		t.Fatalf("unexpected debit row: %#v", preview.Rows[0])
	}
	if preview.Rows[1].Type != 1 || preview.Rows[1].AmountCents != 2550 {
		t.Fatalf("unexpected credit row: %#v", preview.Rows[1])
	}
	if preview.Errors[0].RowNumber != 4 {
		t.Fatalf("expected error on CSV row 4, got %#v", preview.Errors[0])
	}
}
