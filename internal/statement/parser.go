package statement

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"prisma/internal/models"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var supportedDateFormats = map[string]string{
	"yyyy-mm-dd": "2006-01-02",
	"mm/dd/yyyy": "01/02/2006",
	"dd/mm/yyyy": "02/01/2006",
	"mm-dd-yyyy": "01-02-2006",
	"dd-mm-yyyy": "02-01-2006",
}

// Inspect identifies statement columns and returns a small preview.
func Inspect(content string, delimiter string, hasHeader bool) (models.StatementInspection, error) {
	inspection := models.StatementInspection{
		DetectedDateColumn:        -1,
		DetectedDescriptionColumn: -1,
		DetectedAmountColumn:      -1,
		DetectedDebitColumn:       -1,
		DetectedCreditColumn:      -1,
		DetectedDateFormat:        "yyyy-mm-dd",
	}
	records, delimiterName, err := readRecords(content, delimiter)
	if err != nil {
		return inspection, err
	}
	if len(records) == 0 {
		return inspection, fmt.Errorf("the statement file is empty")
	}

	columnCount := len(records[0])
	if columnCount < 2 {
		return inspection, fmt.Errorf("the statement must contain at least two columns")
	}
	inspection.Delimiter = delimiterName
	dataStart := 0
	if hasHeader {
		inspection.Headers = normalizeHeaders(records[0])
		dataStart = 1
	} else {
		inspection.Headers = make([]string, columnCount)
		for index := range inspection.Headers {
			inspection.Headers[index] = fmt.Sprintf("Column %d", index+1)
		}
	}
	if dataStart >= len(records) {
		return inspection, fmt.Errorf("the statement contains a header but no data rows")
	}

	lastSample := dataStart + 5
	if lastSample > len(records) {
		lastSample = len(records)
	}
	inspection.SampleRows = records[dataStart:lastSample]
	inspection.DetectedDateColumn = detectColumn(inspection.Headers, []string{"date", "transactiondate", "posteddate", "valuedate"})
	inspection.DetectedDescriptionColumn = detectColumn(inspection.Headers, []string{"description", "memo", "details", "narrative", "transaction", "payee"})
	inspection.DetectedAmountColumn = detectColumn(inspection.Headers, []string{"amount", "value", "transactionamount"})
	inspection.DetectedDebitColumn = detectColumn(inspection.Headers, []string{"debit", "withdrawal", "moneyout", "outflow"})
	inspection.DetectedCreditColumn = detectColumn(inspection.Headers, []string{"credit", "deposit", "moneyin", "inflow"})
	if inspection.DetectedDateColumn >= 0 && inspection.DetectedDateColumn < len(records[dataStart]) {
		inspection.DetectedDateFormat = detectDateFormat(records[dataStart][inspection.DetectedDateColumn])
	}
	return inspection, nil
}

// Parse converts CSV rows into exact-cent statement entries.
func Parse(content string, options models.StatementParseOptions) (models.StatementPreview, error) {
	preview := models.StatementPreview{Rows: []models.StatementEntry{}, Errors: []models.StatementRowError{}}
	records, _, err := readRecords(content, options.Delimiter)
	if err != nil {
		return preview, err
	}
	if err := validateOptions(options); err != nil {
		return preview, err
	}

	dataStart := 0
	if options.HasHeader {
		dataStart = 1
	}
	occurrences := make(map[string]int)
	for index := dataStart; index < len(records); index++ {
		rowNumber := index + 1
		record := records[index]
		entry, err := parseRecord(record, rowNumber, options)
		if err != nil {
			preview.Errors = append(preview.Errors, models.StatementRowError{RowNumber: rowNumber, Message: err.Error()})
			continue
		}
		baseKey := fmt.Sprintf("%s\x00%s\x00%d\x00%d", entry.Date, strings.ToLower(entry.Description), entry.Type, entry.AmountCents)
		occurrences[baseKey]++
		entry.Occurrence = occurrences[baseKey]
		entry.Fingerprint = Fingerprint(entry.Date, entry.Description, entry.Type, entry.AmountCents, entry.Occurrence)
		preview.Rows = append(preview.Rows, entry)
	}
	if len(preview.Rows) == 0 && len(preview.Errors) == 0 {
		return preview, fmt.Errorf("the statement contains no data rows")
	}
	return preview, nil
}

// Fingerprint creates a stable identifier while preserving legitimate repeated rows.
func Fingerprint(date string, description string, transactionType int, amountCents int64, occurrence int) string {
	value := fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%d", date, strings.ToLower(strings.TrimSpace(description)), transactionType, amountCents, occurrence)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func readRecords(content string, delimiter string) ([][]string, string, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	if strings.TrimSpace(content) == "" {
		return nil, "", fmt.Errorf("the statement file is empty")
	}
	delimiterRune, delimiterName, err := resolveDelimiter(content, delimiter)
	if err != nil {
		return nil, "", err
	}
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = delimiterRune
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = false

	var records [][]string
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, "", fmt.Errorf("could not parse the statement CSV: %w", readErr)
		}
		if isEmptyRecord(record) {
			continue
		}
		records = append(records, record)
	}
	return records, delimiterName, nil
}

func resolveDelimiter(content string, delimiter string) (rune, string, error) {
	options := map[string]rune{"comma": ',', "semicolon": ';', "tab": '\t'}
	if delimiter != "" && delimiter != "auto" {
		value, exists := options[delimiter]
		if !exists {
			return 0, "", fmt.Errorf("unsupported statement delimiter: %s", delimiter)
		}
		return value, delimiter, nil
	}

	firstLine := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")[0]
	bestName := "comma"
	bestCount := -1
	for name, candidate := range options {
		count := countOutsideQuotes(firstLine, candidate)
		if count > bestCount {
			bestName = name
			bestCount = count
		}
	}
	return options[bestName], bestName, nil
}

func countOutsideQuotes(value string, delimiter rune) int {
	count := 0
	inQuotes := false
	for _, character := range value {
		if character == '"' {
			inQuotes = !inQuotes
		} else if character == delimiter && !inQuotes {
			count++
		}
	}
	return count
}

func normalizeHeaders(headers []string) []string {
	result := make([]string, len(headers))
	seen := make(map[string]int)
	for index, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			header = fmt.Sprintf("Column %d", index+1)
		}
		seen[header]++
		if seen[header] > 1 {
			header = fmt.Sprintf("%s (%d)", header, seen[header])
		}
		result[index] = header
	}
	return result
}

func detectColumn(headers []string, candidates []string) int {
	for index, header := range headers {
		normalized := normalizeHeader(header)
		for _, candidate := range candidates {
			if normalized == candidate {
				return index
			}
		}
	}
	return -1
}

func normalizeHeader(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			return unicode.ToLower(character)
		}
		return -1
	}, value)
}

func detectDateFormat(value string) string {
	value = strings.TrimSpace(value)
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return "yyyy-mm-dd"
	}
	parts := strings.FieldsFunc(value, func(character rune) bool { return character == '/' || character == '-' })
	if len(parts) != 3 {
		return "yyyy-mm-dd"
	}
	first, _ := strconv.Atoi(parts[0])
	second, _ := strconv.Atoi(parts[1])
	separator := "/"
	if strings.Contains(value, "-") {
		separator = "-"
	}
	if first > 12 {
		return "dd" + separator + "mm" + separator + "yyyy"
	}
	if second > 12 {
		return "mm" + separator + "dd" + separator + "yyyy"
	}
	return "mm" + separator + "dd" + separator + "yyyy"
}

func validateOptions(options models.StatementParseOptions) error {
	if options.DateColumn < 0 || options.DescriptionColumn < 0 {
		return fmt.Errorf("date and description columns are required")
	}
	if _, exists := supportedDateFormats[options.DateFormat]; !exists {
		return fmt.Errorf("unsupported date format: %s", options.DateFormat)
	}
	if options.DecimalSeparator != "auto" && options.DecimalSeparator != "dot" && options.DecimalSeparator != "comma" {
		return fmt.Errorf("unsupported decimal separator: %s", options.DecimalSeparator)
	}
	if options.AmountMode == "signed" && options.AmountColumn < 0 {
		return fmt.Errorf("an amount column is required")
	}
	if options.AmountMode == "debit_credit" && options.DebitColumn < 0 && options.CreditColumn < 0 {
		return fmt.Errorf("a debit or credit column is required")
	}
	if options.AmountMode != "signed" && options.AmountMode != "debit_credit" {
		return fmt.Errorf("unsupported amount mode: %s", options.AmountMode)
	}
	return nil
}

func parseRecord(record []string, rowNumber int, options models.StatementParseOptions) (models.StatementEntry, error) {
	entry := models.StatementEntry{RowNumber: rowNumber}
	requiredIndexes := []int{options.DateColumn, options.DescriptionColumn}
	if options.AmountMode == "signed" {
		requiredIndexes = append(requiredIndexes, options.AmountColumn)
	} else {
		requiredIndexes = append(requiredIndexes, options.DebitColumn, options.CreditColumn)
	}
	for _, index := range requiredIndexes {
		if index >= len(record) {
			return entry, fmt.Errorf("the row has fewer columns than expected")
		}
	}

	parsedDate, err := time.Parse(supportedDateFormats[options.DateFormat], strings.TrimSpace(record[options.DateColumn]))
	if err != nil {
		return entry, fmt.Errorf("invalid date")
	}
	entry.Date = parsedDate.Format("2006-01-02")
	entry.Description = strings.TrimSpace(record[options.DescriptionColumn])
	if entry.Description == "" {
		return entry, fmt.Errorf("description is empty")
	}

	var signedAmount int64
	if options.AmountMode == "signed" {
		signedAmount, err = parseAmount(record[options.AmountColumn], options.DecimalSeparator, true)
	} else {
		var debitAmount int64
		var creditAmount int64
		if options.DebitColumn >= 0 {
			debitAmount, err = parseAmount(record[options.DebitColumn], options.DecimalSeparator, false)
			if err != nil {
				return entry, fmt.Errorf("invalid debit amount")
			}
		}
		if options.CreditColumn >= 0 {
			creditAmount, err = parseAmount(record[options.CreditColumn], options.DecimalSeparator, false)
			if err != nil {
				return entry, fmt.Errorf("invalid credit amount")
			}
		}
		if debitAmount != 0 && creditAmount != 0 {
			return entry, fmt.Errorf("both debit and credit contain an amount")
		}
		if debitAmount != 0 {
			signedAmount = -debitAmount
		} else {
			signedAmount = creditAmount
		}
	}
	if err != nil || signedAmount == 0 {
		return entry, fmt.Errorf("invalid or zero amount")
	}
	if signedAmount < 0 {
		entry.Type = -1
		entry.AmountCents = -signedAmount
	} else {
		entry.Type = 1
		entry.AmountCents = signedAmount
	}
	if entry.AmountCents > models.MaxSafeAmountCents {
		return entry, fmt.Errorf("amount exceeds the maximum exact supported value")
	}
	return entry, nil
}

func parseAmount(value string, decimalSeparator string, allowSign bool) (int64, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\u00a0", ""))
	if value == "" {
		return 0, nil
	}
	negative := strings.HasPrefix(value, "-") || (strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")"))
	if negative && !allowSign {
		return 0, fmt.Errorf("negative value is not allowed in separate debit or credit columns")
	}

	var cleaned strings.Builder
	for _, character := range value {
		if unicode.IsDigit(character) || character == '.' || character == ',' {
			cleaned.WriteRune(character)
		}
	}
	numeric := cleaned.String()
	if numeric == "" {
		return 0, fmt.Errorf("amount does not contain digits")
	}
	separator := determineDecimalSeparator(numeric, decimalSeparator)
	if separator != 0 {
		lastIndex := strings.LastIndex(numeric, string(separator))
		whole := removeSeparators(numeric[:lastIndex])
		fraction := removeSeparators(numeric[lastIndex+1:])
		if len(fraction) == 0 || len(fraction) > 2 {
			return 0, fmt.Errorf("amount must contain at most two decimal places")
		}
		numeric = whole + "." + fraction
	} else {
		numeric = removeSeparators(numeric)
	}
	parts := strings.Split(numeric, ".")
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	cents, err := strconv.ParseInt(strings.TrimLeft(whole+fraction, "0"), 10, 64)
	if err != nil {
		if strings.Trim(whole+fraction, "0") == "" {
			return 0, nil
		}
		return 0, err
	}
	if negative {
		cents = -cents
	}
	return cents, nil
}

func determineDecimalSeparator(value string, preference string) rune {
	if preference == "dot" {
		return '.'
	}
	if preference == "comma" {
		return ','
	}
	lastDot := strings.LastIndex(value, ".")
	lastComma := strings.LastIndex(value, ",")
	if lastDot >= 0 && lastComma >= 0 {
		if lastDot > lastComma {
			return '.'
		}
		return ','
	}
	index := lastDot
	separator := '.'
	if lastComma >= 0 {
		index = lastComma
		separator = ','
	}
	if index >= 0 && len(value)-index-1 <= 2 {
		return separator
	}
	return 0
}

func removeSeparators(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsDigit(character) {
			return character
		}
		return -1
	}, value)
}

func isEmptyRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
