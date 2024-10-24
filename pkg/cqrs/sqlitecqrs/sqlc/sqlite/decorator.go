package sqlc

import (
	"strings"

	"github.com/oklog/ulid/v2"
)

// EventIDs convert the blob data to a list of ULIDs
func (e EventBatch) EventIDs() ([]ulid.ULID, error) {
	strids := strings.Split(string(e.EventIds), ",")
	ids := make([]ulid.ULID, len(strids))

	for i, sid := range strids {
		id, err := ulid.Parse(sid)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}
