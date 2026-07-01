package util

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ────────────────────────────────────────────────
// ParseTime
// ────────────────────────────────────────────────

func TestParseTime(t *testing.T) {
	t.Run("RFC3339 format", func(t *testing.T) {
		result := ParseTime("2026-04-09T10:00:00Z")
		assert.Equal(t, 2026, result.Year())
		assert.Equal(t, time.April, result.Month())
		assert.Equal(t, 9, result.Day())
	})

	t.Run("YYYY-MM-DD format", func(t *testing.T) {
		result := ParseTime("2026-04-09")
		assert.Equal(t, 2026, result.Year())
		assert.Equal(t, time.April, result.Month())
	})

	t.Run("empty string returns zero time", func(t *testing.T) {
		result := ParseTime("")
		assert.True(t, result.IsZero())
	})

	t.Run("invalid string returns zero time", func(t *testing.T) {
		result := ParseTime("not-a-date")
		assert.True(t, result.IsZero())
	})
}

// ────────────────────────────────────────────────
// StringToTimePtr
// ────────────────────────────────────────────────

func TestStringToTimePtr(t *testing.T) {
	t.Run("valid RFC3339 string returns pointer", func(t *testing.T) {
		result := StringToTimePtr("2026-04-09T10:00:00Z")
		require.NotNil(t, result)
		assert.Equal(t, 2026, result.Year())
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		result := StringToTimePtr("")
		assert.Nil(t, result)
	})

	t.Run("invalid string returns nil", func(t *testing.T) {
		result := StringToTimePtr("garbage")
		assert.Nil(t, result)
	})
}

// ────────────────────────────────────────────────
// PtrStringToTimePtr
// ────────────────────────────────────────────────

func TestPtrStringToTimePtr(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := PtrStringToTimePtr(nil)
		assert.Nil(t, result)
	})

	t.Run("valid string pointer returns time pointer", func(t *testing.T) {
		s := "2026-04-09T10:00:00Z"
		result := PtrStringToTimePtr(&s)
		require.NotNil(t, result)
		assert.Equal(t, 2026, result.Year())
	})
}

// ────────────────────────────────────────────────
// StringToPtr
// ────────────────────────────────────────────────

func TestStringToPtr(t *testing.T) {
	t.Run("non-empty string returns pointer", func(t *testing.T) {
		result := StringToPtr("hello")
		require.NotNil(t, result)
		assert.Equal(t, "hello", *result)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		result := StringToPtr("")
		assert.Nil(t, result)
	})
}

// ────────────────────────────────────────────────
// FloatToPgNumeric / PtrFloatToPgNumeric
// ────────────────────────────────────────────────

func TestFloatToPgNumeric(t *testing.T) {
	t.Run("non-zero value is valid", func(t *testing.T) {
		result := FloatToPgNumeric(42.5)
		assert.True(t, result.Valid)
	})

	t.Run("zero value returns invalid numeric", func(t *testing.T) {
		result := FloatToPgNumeric(0)
		assert.False(t, result.Valid)
	})
}

func TestPtrFloatToPgNumeric(t *testing.T) {
	t.Run("nil pointer returns invalid numeric", func(t *testing.T) {
		result := PtrFloatToPgNumeric(nil)
		assert.False(t, result.Valid)
	})

	t.Run("non-nil pointer returns valid numeric", func(t *testing.T) {
		f := 99.9
		result := PtrFloatToPgNumeric(&f)
		assert.True(t, result.Valid)
	})
}

// ────────────────────────────────────────────────
// PgNumericToFloat64Ptr
// ────────────────────────────────────────────────

func TestPgNumericToFloat64Ptr(t *testing.T) {
	t.Run("invalid numeric returns nil", func(t *testing.T) {
		result := PgNumericToFloat64Ptr(pgtype.Numeric{Valid: false})
		assert.Nil(t, result)
	})

	t.Run("NaN returns nil", func(t *testing.T) {
		result := PgNumericToFloat64Ptr(pgtype.Numeric{Valid: true, NaN: true})
		assert.Nil(t, result)
	})

	t.Run("valid numeric with no exponent roundtrips correctly", func(t *testing.T) {
		// Build via FloatToPgNumeric then convert back.
		n := FloatToPgNumeric(42.0)
		result := PgNumericToFloat64Ptr(n)
		require.NotNil(t, result)
		assert.InDelta(t, 42.0, *result, 0.001)
	})
}

// ────────────────────────────────────────────────
// TimestamptzToTimePtr
// ────────────────────────────────────────────────

func TestTimestamptzToTimePtr(t *testing.T) {
	t.Run("invalid timestamptz returns nil", func(t *testing.T) {
		result := TimestamptzToTimePtr(pgtype.Timestamptz{Valid: false})
		assert.Nil(t, result)
	})

	t.Run("valid timestamptz returns time pointer", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		result := TimestamptzToTimePtr(pgtype.Timestamptz{Time: now, Valid: true})
		require.NotNil(t, result)
		assert.Equal(t, now, *result)
	})
}

// ────────────────────────────────────────────────
// TimeToTimestamptz
// ────────────────────────────────────────────────

func TestTimeToTimestamptz(t *testing.T) {
	t.Run("nil time returns invalid timestamptz", func(t *testing.T) {
		result := TimeToTimestamptz(nil)
		assert.False(t, result.Valid)
	})

	t.Run("non-nil time returns valid timestamptz", func(t *testing.T) {
		now := time.Now().UTC()
		result := TimeToTimestamptz(&now)
		assert.True(t, result.Valid)
		assert.Equal(t, now, result.Time)
	})
}
