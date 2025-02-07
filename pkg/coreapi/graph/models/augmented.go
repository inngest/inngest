package models

type WorkerConnectionsConnection struct {
	Edges    []*ConnectV1WorkerConnectionEdge `json:"edges"`
	PageInfo *PageInfo                        `json:"pageInfo"`

	After   *string
	Filter  ConnectV1WorkerConnectionsFilter
	OrderBy []*ConnectV1WorkerConnectionsOrderBy
}

type RunsV2Connection struct {
	Edges    []*FunctionRunV2Edge `json:"edges"`
	PageInfo *PageInfo            `json:"pageInfo"`
 
	After   *string
	Filter  RunsFilterV2
	OrderBy []*RunsV2OrderBy
}
