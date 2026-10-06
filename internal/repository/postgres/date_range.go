package postgres

import (
	"time"
)

// ParseDateRangeUTC parses a "YYYY-MM-DD" date string and returns the start of the day (inclusive)
// and the start of the next day (exclusive) in UTC.
func ParseDateRangeUTC(date string) (time.Time, time.Time, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	return start, end, nil
}
