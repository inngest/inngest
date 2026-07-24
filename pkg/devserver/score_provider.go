package devserver

import (
	"context"
	"database/sql"
	"errors"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
)

func scoreMetadataLoader(data runProviderDataReader) apiv2.MissingScoreMetadataLoader {
	return func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error) {
		fnrun, err := data.GetFunctionRun(ctx, id.Tenant.AccountID, id.Tenant.EnvID, id.RunID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, statev2.ErrMetadataNotFound
			}
			return nil, err
		}

		fn, err := data.GetFunctionByInternalUUID(ctx, fnrun.FunctionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, statev2.ErrMetadataNotFound
			}
			return nil, err
		}

		md := &statev2.Metadata{
			ID: statev2.ID{
				RunID:      fnrun.RunID,
				FunctionID: fnrun.FunctionID,
				Tenant: statev2.Tenant{
					AccountID: id.Tenant.AccountID,
					EnvID:     id.Tenant.EnvID,
					AppID:     fn.AppID,
				},
			},
			Config: statev2.Config{
				FunctionVersion: int(fnrun.FunctionVersion),
				BatchID:         fnrun.BatchID,
				StartedAt:       fnrun.RunStartedAt,
				EventIDs:        []ulid.ULID{fnrun.EventID},
				OriginalRunID:   fnrun.OriginalRunID,
			},
		}
		statev2.InitConfig(&md.Config)
		return md, nil
	}
}
