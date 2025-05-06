package cqrs

import (
	"context"

	"github.com/google/uuid"
)

type EventTypeManager interface {
	EventTypeWriter
	EventTypeReader
}

type EventTypeWriter interface{}

type EventTypeReader interface {
	GetEventTypes(
		ctx context.Context,
		filter EventTypesFilter,
	) (*EventTypesConnection, error)
}

type EventType struct {
	EnvID uuid.UUID `json:"envID"`
	Name  string    `json:"name"`
}

type EventTypesFilter struct {
	Archived   *bool   `json:"archived,omitempty"`
	NameSearch *string `json:"nameSearch,omitempty"`
}

type EventTypesConnection struct {
	Edges      []*EventTypesEdge `json:"edges"`
	PageInfo   *PageInfo         `json:"pageInfo"`
	TotalCount int               `json:"totalCount"`
}

type EventTypesEdge struct {
	Cursor string     `json:"cursor"`
	Node   *EventType `json:"node"`
}
