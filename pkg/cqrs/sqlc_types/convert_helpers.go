package sqlc_types

import (
	"database/sql"
	"time"
)

func toNullStringFromString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullInt64FromInt32(v int32) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(v), Valid: true}
}

func toNullInt64FromNullInt32(v sql.NullInt32) sql.NullInt64 {
	if !v.Valid {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(v.Int32), Valid: true}
}

func toNullTimeFromTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
