package apiv2

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apiv1 "github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
)

// ScoreAuthResolver returns the account and workspace a score should be
// recorded under for the current request.
type ScoreAuthResolver func(ctx context.Context) (accountID, envID uuid.UUID, err error)

// ScoresEnabledFlag reports whether score submission is enabled for an account.
type ScoresEnabledFlag func(ctx context.Context, accountID uuid.UUID) bool

type ScoreStepTargetValidatorParams struct {
	RunID       ulid.ULID
	AccountID   uuid.UUID
	EnvID       uuid.UUID
	StepID      string
	TraceStepID string
}

// ScoreStepTargetValidator validates step-scoped score targets when live state
// cannot prove that the step exists.
type ScoreStepTargetValidator func(ctx context.Context, params ScoreStepTargetValidatorParams) error

type MissingScoreMetadataLoader func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error)

type StateScoreProviderOptions struct {
	State              statev2.RunService
	TracerProvider     tracing.TracerProvider
	TraceReader        cqrs.TraceReader
	Auth               ScoreAuthResolver
	Enabled            ScoresEnabledFlag
	MissingStateLoader MissingScoreMetadataLoader
	StepValidator      ScoreStepTargetValidator
}

// NewStateScoreProvider returns a ScoreProvider that records scores as
// inngest.score metadata
func NewStateScoreProvider(opts StateScoreProviderOptions) ScoreProvider {
	return &stateScoreProvider{
		state:              opts.State,
		tracerProvider:     opts.TracerProvider,
		traceReader:        opts.TraceReader,
		auth:               opts.Auth,
		enabled:            opts.Enabled,
		missingStateLoader: opts.MissingStateLoader,
		stepValidator:      opts.StepValidator,
	}
}

type stateScoreProvider struct {
	state              statev2.RunService
	tracerProvider     tracing.TracerProvider
	traceReader        cqrs.TraceReader
	auth               ScoreAuthResolver
	enabled            ScoresEnabledFlag
	missingStateLoader MissingScoreMetadataLoader
	stepValidator      ScoreStepTargetValidator
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

	stateID := statev2.ID{
		RunID: params.RunID,
		Tenant: statev2.Tenant{
			EnvID:     envID,
			AccountID: accountID,
		},
	}

	if err := p.validateStepTargets(ctx, stateID, params.Scores); err != nil {
		return err
	}

	for _, score := range params.Scores {
		updates, err := scoreMetadataUpdates(score)
		if err != nil {
			return err
		}
		err = apiv1.AddRunMetadata(ctx, apiv1.AddRunMetadataOpts{
			State:          p.metadataState(),
			TracerProvider: p.tracerProvider,
			TraceReader:    p.traceReader,
		}, scoreAuth{accountID: accountID, envID: envID}, params.RunID, &apiv1.AddRunMetadataRequest{
			Target: apiv1.RunMetadataTarget{
				StepID: score.StepID,
			},
			Metadata: updates,
		})
		switch {
		case errors.Is(err, statev2.ErrRunNotFound), errors.Is(err, statev2.ErrMetadataNotFound):
			return fmt.Errorf("%w: %s", ErrScoreTargetNotFound, params.RunID)
		case err != nil:
			return err
		}
	}

	return nil
}

func scoreMetadataUpdates(score ScoreInput) ([]metadata.Update, error) {
	updates := make([]metadata.Update, 0, 2)
	if score.Experiment != nil {
		update, err := ScoreExperimentMetadataUpdate(*score.Experiment)
		if err != nil {
			return nil, err
		}
		updates = append(updates, update)
	}

	update, err := ScoreMetadataUpdate(score.Name, score.Value)
	if err != nil {
		return nil, err
	}
	updates = append(updates, update)
	return updates, nil
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

func (p *stateScoreProvider) validateStepTargets(ctx context.Context, id statev2.ID, scores []ScoreInput) error {
	stepIDs := map[string]struct{}{}
	for _, score := range scores {
		if score.StepID != nil {
			stepIDs[*score.StepID] = struct{}{}
		}
	}
	if len(stepIDs) == 0 {
		return nil
	}

	var md *statev2.Metadata
	if p.state != nil {
		loaded, err := p.state.LoadMetadata(ctx, id)
		if err == nil {
			md = &loaded
		} else if !errors.Is(err, statev2.ErrRunNotFound) && !errors.Is(err, statev2.ErrMetadataNotFound) {
			return err
		}
	}

	for stepID := range stepIDs {
		if md != nil && scoreStepExistsInState(md, stepID) {
			continue
		}
		if p.stepValidator == nil {
			if md == nil {
				continue
			}
			return fmt.Errorf("%w: %s", ErrScoreTargetNotFound, stepID)
		}
		if err := p.stepValidator(ctx, ScoreStepTargetValidatorParams{
			RunID:       id.RunID,
			AccountID:   id.Tenant.AccountID,
			EnvID:       id.Tenant.EnvID,
			StepID:      stepID,
			TraceStepID: scoreTraceStepID(stepID),
		}); err != nil {
			if errors.Is(err, ErrScoreTargetNotFound) {
				return fmt.Errorf("%w: %s", ErrScoreTargetNotFound, stepID)
			}
			return err
		}
	}

	return nil
}

func scoreStepExistsInState(md *statev2.Metadata, stepID string) bool {
	traceStepID := scoreTraceStepID(stepID)
	for _, candidate := range md.Stack {
		// Older callers may send raw SDK step IDs, while finalized state stores
		// step spans under their trace IDs.
		if candidate == stepID || candidate == traceStepID {
			return true
		}
	}
	return false
}

func scoreTraceStepID(stepID string) string {
	sum := sha1.Sum([]byte(stepID))
	return hex.EncodeToString(sum[:])
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
