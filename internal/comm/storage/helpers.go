package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// timeNow returns the current UTC time
func timeNow() time.Time {
	return time.Now().UTC()
}

// derefInt64 safely dereferences *int64, returning def if nil.
func derefInt64(p *int64, def int64) int64 {
	if p == nil {
		return def
	}
	return *p
}

// derefString safely dereferences *string, returning def if nil.
func derefString(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

// derefFloat64 safely dereferences *float64, returning def if nil.
func derefFloat64(p *float64, def float64) float64 {
	if p == nil {
		return def
	}
	return *p
}

// int64Ptr creates a pointer from an int64 value.
func int64Ptr(v int64) *int64 {
	return &v
}

// stringPtr creates a pointer from a string value (nil if empty).
func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// timeToUnixPtr converts *time.Time to *int64 (Unix timestamp).
func timeToUnixPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	v := t.Unix()
	return &v
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func timeToUnix(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}

func unixToTime(t sql.NullInt64) *time.Time {
	if !t.Valid {
		return nil
	}
	u := time.Unix(t.Int64, 0).UTC()
	return &u
}

func boolToNullInt64(b bool) sql.NullInt64 {
	if b {
		return sql.NullInt64{Int64: 1, Valid: true}
	}
	return sql.NullInt64{Int64: 0, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
