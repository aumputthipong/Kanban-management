package util

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ParseTime parses an ISO 8601 or YYYY-MM-DD string, returning the zero time on failure.
func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02", s)
	}
	return t
}

// StringToTimePtr parses s to *time.Time, returning nil if empty or unparseable.
func StringToTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := ParseTime(s)
	if t.IsZero() {
		return nil
	}
	return &t
}

// PtrStringToTimePtr converts *string to *time.Time.
func PtrStringToTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return StringToTimePtr(*s)
}

// StringToPtr returns nil for an empty string, otherwise a pointer to it.
func StringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// FloatToPgNumeric converts a float64 to pgtype.Numeric (budget, estimated_hours).
func FloatToPgNumeric(f float64) pgtype.Numeric {
	if f == 0 {
		return pgtype.Numeric{Valid: false}
	}
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%f", f))
	return n
}

// PtrFloatToPgNumeric converts *float64 to pgtype.Numeric.
func PtrFloatToPgNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%f", *f))
	return n
}

// PgNumericToFloat64Ptr converts pgtype.Numeric to *float64, returning nil if invalid.
func PgNumericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid || n.NaN || n.Int == nil {
		return nil
	}
	text := fmt.Sprintf("%d", n.Int)
	base, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil
	}
	if n.Exp != 0 {
		base = base * math.Pow10(int(n.Exp))
	}
	return &base
}

// TimestamptzToTimePtr converts a pgtype.Timestamptz read from the DB to *time.Time.
func TimestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// TimeToTimestamptz converts *time.Time to pgtype.Timestamptz for TIMESTAMPTZ columns.
func TimeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
