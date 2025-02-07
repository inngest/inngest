package models

type WorkerConnectionsConnection struct {
	Edges    []*ConnectV1WorkerConnectionEdge `json:"edges"`
	PageInfo *PageInfo                        `json:"pageInfo"`

	After   *string
	Filter  ConnectV1WorkerConnectionsFilter
	OrderBy []*ConnectV1WorkerConnectionsOrderBy
}
