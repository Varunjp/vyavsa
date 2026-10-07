package postgres_test

import (
	"testing"

	"github.com/Varunjp/vyavsa/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
)

func TestMaskAccountNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", "Active Account"},
		{"Spaces only", "   ", "Active Account"},
		{"Short account 3 digits", "123", "****123"},
		{"4 digits account", "1234", "****1234"},
		{"Standard 11 digit account", "12345678901", "****8901"},
		{"Account with spaces", " 9876543210 ", "****3210"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := postgres.MaskAccountNumber(tt.input)
			assert.Equal(t, tt.expected, res)
		})
	}
}
