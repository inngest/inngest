package apiv2

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/metadatawriter"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
)

const scorePkgName = "apiv2.inngest"

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

type StateScoreProviderOptions struct {
	State              statev2.RunService
	TracerProvider     tracing.TracerProvider
	Auth               ScoreAuthResolver
	Enabled            ScoresEnabledFlag
	MissingStateLoader metadatawriter.MissingStateLoader
	StepValidator      ScoreStepTargetValidator
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
		stepValidator:      opts.StepValidator,
	}
}

type stateScoreProvider struct {
	state              statev2.RunService
	tracerProvider     tracing.TracerProvider
	auth               ScoreAuthResolver
	enabled            ScoresEnabledFlag
	missingStateLoader metadatawriter.MissingStateLoader
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

	items := make([]metadatawriter.Item, 0, len(params.Scores)*2)
	for _, score := range params.Scores {
		if len(score.Metadata) == 0 {
			return errors.New("score metadata is required")
		}
		stepID := score.StepID
		for _, update := range score.Metadata {
			items = append(items, metadatawriter.Item{
				Metadata: update,
				Parent: func(md *statev2.Metadata) (*meta.SpanReference, metadata.Scope) {
					return scoreParentSpanRef(md, stepID)
				},
			})
		}
	}

	writer := metadatawriter.Writer{
		State:              p.state,
		TracerProvider:     p.tracerProvider,
		MissingStateLoader: p.missingStateLoader,
		PkgName:            scorePkgName,
	}

	err = writer.Write(ctx, metadatawriter.WriteRequest{
		ID:       stateID,
		Items:    items,
		Location: "Service.CreateScore",
	})
	switch {
	case errors.Is(err, statev2.ErrRunNotFound), errors.Is(err, statev2.ErrMetadataNotFound):
		return fmt.Errorf("%w: %s", ErrScoreTargetNotFound, params.RunID)
	case err != nil:
		return err
	}

	return nil
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
	_, ok := scoreStepIDForSpan(md, stepID)
	return ok
}

func scoreStepIDForSpan(md *statev2.Metadata, stepID string) (string, bool) {
	traceStepID := scoreTraceStepID(stepID)
	for _, candidate := range md.Stack {
		// live run state contains raw sdk step ids while finalized trace lookup
		// stores step spans under hashed trace step ids.
		if candidate == stepID && !isSHA1Hex(stepID) {
			return candidate, true
		}
		if candidate == traceStepID {
			return traceStepID, true
		}
	}
	return "", false
}

func scoreParentSpanRef(stateMetadata *statev2.Metadata, stepID *string) (*meta.SpanReference, metadata.Scope) {
	if stepID == nil {
		return tracing.RunSpanRefFromMetadata(stateMetadata), enums.MetadataScopeRun
	}

	spanStepID := scoreTraceStepID(*stepID)
	if matchedStepID, ok := scoreStepIDForSpan(stateMetadata, *stepID); ok {
		spanStepID = matchedStepID
	}
	return tracing.FinalizedStepSpanRefFromMetadataAndStepID(stateMetadata, spanStepID), enums.MetadataScopeStep
}

func scoreTraceStepID(stepID string) string {
	sum := sha1.Sum([]byte(stepID))
	return hex.EncodeToString(sum[:])
}

func isSHA1Hex(value string) bool {
	if len(value) != sha1.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
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
