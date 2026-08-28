package models

// Category represents a transaction column and its balance direction.
type Category struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Type   int    `json:"type"` // 1 for Income, -1 for Expense
	Active bool   `json:"active"`
}
