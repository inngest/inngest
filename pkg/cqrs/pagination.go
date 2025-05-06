package cqrs

type PageInfo struct {
	EndCursor       *string `json:"endCursor,omitempty"`
	HasNextPage     bool    `json:"hasNextPage"`
	HasPreviousPage bool    `json:"hasPreviousPage"`
	StartCursor     *string `json:"startCursor,omitempty"`
}
