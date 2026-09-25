package util

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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

func PtrStringToTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return StringToTimePtr(*s)
}

func StringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func FloatToPgNumeric(f float64) pgtype.Numeric {
	if f == 0 {
		return pgtype.Numeric{Valid: false}
	}
	var n pgtype.Numeric
	if err := n.Scan(fmt.Sprintf("%f", f)); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
}

func PtrFloatToPgNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{Valid: false}
	}
	var n pgtype.Numeric
	if err := n.Scan(fmt.Sprintf("%f", *f)); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
}

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

func TimestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func TimeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
