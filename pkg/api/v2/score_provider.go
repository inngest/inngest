package apiv2

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	apiv1 "github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

// ScoreAuthResolver returns the account and workspace a score should be
// recorded under for the current request.
type ScoreAuthResolver func(ctx context.Context) (accountID, envID uuid.UUID, err error)

// ScoresEnabledFlag reports whether score submission is enabled for an account.
type ScoresEnabledFlag func(ctx context.Context, accountID uuid.UUID) bool

type MissingScoreMetadataLoader func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error)

type StateScoreProviderOptions struct {
	State              statev2.RunService
	TracerProvider     tracing.TracerProvider
	Auth               ScoreAuthResolver
	Enabled            ScoresEnabledFlag
	MissingStateLoader MissingScoreMetadataLoader
}

// NewStateScoreProvider returns a ScoreProvider that records scores as
// inngest.score metadata
func NewStateScoreProvider(opts StateScoreProviderOptions) ScoreProvider {
	return &stateScoreProvider{
		state:              opts.State,
		tracerProvider:     opts.TracerProvider,
		auth:               opts.Auth,
		enabled:            opts.Enabled,
		missingStateLoader: opts.MissingStateLoader,
	}
}

type stateScoreProvider struct {
	state              statev2.RunService
	tracerProvider     tracing.TracerProvider
	auth               ScoreAuthResolver
	enabled            ScoresEnabledFlag
	missingStateLoader MissingScoreMetadataLoader
}

func (p *stateScoreProvider) CreateScores(ctx context.Context, params CreateScoresParams) error {
	accountID, envID, err := p.auth(ctx)
	if err != nil {
		return err
	}

	if p.enabled != nil && !p.enabled(ctx, accountID) {
		return ErrScoresNotEnabled
	}

	if p.state == nil {
		return errors.New("score provider state is nil")
	}

	for _, score := range params.Scores {
		if len(score.Metadata) == 0 {
			return errors.New("score metadata is required")
		}
		err = apiv1.AddRunMetadata(ctx, apiv1.AddRunMetadataOpts{
			State:          p.metadataState(),
			TracerProvider: p.tracerProvider,
		}, scoreAuth{accountID: accountID, envID: envID}, params.RunID, &apiv1.AddRunMetadataRequest{
			Target: apiv1.RunMetadataTarget{
				StepID: score.StepID,
			},
			Metadata: score.Metadata,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *stateScoreProvider) metadataState() statev2.RunService {
	if p.missingStateLoader == nil {
		return p.state
	}
	return scoreMetadataState{
		RunService: p.state,
		load:       p.missingStateLoader,
	}
}

type scoreMetadataState struct {
	statev2.RunService
	load MissingScoreMetadataLoader
}

func (s scoreMetadataState) LoadMetadata(ctx context.Context, id statev2.ID, opts ...statev2.LoadMetadataOption) (statev2.Metadata, error) {
	md, err := s.RunService.LoadMetadata(ctx, id, opts...)
	if err == nil {
		return md, nil
	}
	if !errors.Is(err, statev2.ErrRunNotFound) && !errors.Is(err, statev2.ErrMetadataNotFound) {
		return statev2.Metadata{}, err
	}

	loaded, loadErr := s.load(ctx, id)
	if loadErr != nil {
		return statev2.Metadata{}, loadErr
	}
	if loaded == nil {
		return statev2.Metadata{}, err
	}
	return *loaded, nil
}

func (s scoreMetadataState) IncrementMetadataSize(ctx context.Context, id statev2.ID, delta int) error {
	if inc, ok := s.RunService.(statev2.MetadataSizeIncrementer); ok {
		return inc.IncrementMetadataSize(ctx, id, delta)
	}
	return nil
}

type scoreAuth struct {
	accountID uuid.UUID
	envID     uuid.UUID
}

func (s scoreAuth) AccountID() uuid.UUID {
	return s.accountID
}

func (s scoreAuth) WorkspaceID() uuid.UUID {
	return s.envID
}

// ScoreMetadataUpdate builds and validates the metadata update for a named
// score, applying the same rules as SDK score submission.
func ScoreMetadataUpdate(name string, value any) (metadata.Update, error) {
	raw, err := json.Marshal(struct {
		Value any `json:"value"`
	}{
		Value: value,
	})
	if err != nil {
		return metadata.Update{}, err
	}

	update := metadata.Update{
		RawUpdate: metadata.RawUpdate{
			Kind:   metadata.KindInngestScore,
			Op:     enums.MetadataOpcodeMerge,
			Values: metadata.Values{name: raw},
		},
	}
	if err := update.ValidateAllowed(); err != nil {
		return metadata.Update{}, err
	}

	return update, nil
}

func ScoreExperimentMetadataUpdate(experiment ScoreExperimentInput) (metadata.Update, error) {
	name, err := json.Marshal(experiment.ExperimentName)
	if err != nil {
		return metadata.Update{}, err
	}
	variant, err := json.Marshal(experiment.Variant)
	if err != nil {
		return metadata.Update{}, err
	}

	update := metadata.Update{
		RawUpdate: metadata.RawUpdate{
			Kind: metadata.KindInngestExperiment,
			Op:   enums.MetadataOpcodeMerge,
			Values: metadata.Values{
				"name":    name,
				"variant": variant,
			},
		},
	}
	if err := update.ValidateAllowed(); err != nil {
		return metadata.Update{}, err
	}

	return update, nil
}
