package coreapi

import (
	"database/sql"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalNullString is a custom marshaller.
func MarshalNullString(ns sql.NullString) graphql.Marshaler {
	if !ns.Valid {
		// this is also important, so we can detect if this scalar is used in a not null context and return an appropriate error
		return graphql.Null
	}
	return graphql.MarshalString(ns.String)
}

// UnmarshalNullString is a custom unmarshaller.
func UnmarshalNullString(v interface{}) (sql.NullString, error) {
	if v == nil {
		return sql.NullString{Valid: false}, nil
	}
	// again you can delegate to the default implementation to save yourself some work.
	s, err := graphql.UnmarshalString(v)
	return sql.NullString{String: s, Valid: true}, err
}
