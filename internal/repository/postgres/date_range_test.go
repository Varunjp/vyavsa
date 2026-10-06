package postgres_test

import (
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateRangeUTC(t *testing.T) {
	t.Run("Valid date returns midnight UTC start and start of next day", func(t *testing.T) {
		start, end, err := postgres.ParseDateRangeUTC("2026-10-06")
		require.NoError(t, err)

		expectedStart := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
		expectedEnd := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)

		assert.Equal(t, expectedStart, start)
		assert.Equal(t, expectedEnd, end)
		assert.Equal(t, 24*time.Hour, end.Sub(start))
	})

	t.Run("Invalid date returns error", func(t *testing.T) {
		_, _, err := postgres.ParseDateRangeUTC("invalid-date")
		assert.Error(t, err)

		_, _, err = postgres.ParseDateRangeUTC("06-10-2026")
		assert.Error(t, err)
	})
}
