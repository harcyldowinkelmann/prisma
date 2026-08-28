package main

import "testing"

func TestParseAmountToCents(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		cents int64
	}{
		{name: "whole units", input: "125", cents: 12500},
		{name: "one decimal place", input: "125.4", cents: 12540},
		{name: "two decimal places", input: "125.45", cents: 12545},
		{name: "leading decimal point", input: ".05", cents: 5},
		{name: "leading zeros", input: "0001.20", cents: 120},
		{name: "surrounding spaces", input: " 0.01 ", cents: 1},
		{name: "largest supported value", input: "90071992547409.91", cents: 9007199254740991},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cents, err := parseAmountToCents(testCase.input)
			if err != nil {
				t.Fatalf("parse amount: %v", err)
			}
			if cents != testCase.cents {
				t.Fatalf("expected %d cents, got %d", testCase.cents, cents)
			}
		})
	}
}

func TestParseAmountToCentsRejectsInvalidValues(t *testing.T) {
	invalidValues := []string{
		"",
		"0",
		"0.00",
		"-1.00",
		"1.001",
		"1,00",
		"one",
		"1.2.3",
		"90071992547409.92",
	}

	for _, value := range invalidValues {
		t.Run(value, func(t *testing.T) {
			if _, err := parseAmountToCents(value); err == nil {
				t.Fatal("expected invalid amount to be rejected")
			}
		})
	}
}
