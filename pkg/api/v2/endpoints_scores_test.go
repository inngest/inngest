package apiv2

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeScoreProvider struct {
	err    error
	params *CreateScoresParams
}

func (f *fakeScoreProvider) CreateScores(ctx context.Context, params CreateScoresParams) error {
	f.params = &params
	return f.err
}

func TestCreateScore(t *testing.T) {
	runID := ulid.MustParse("01HP1ZX8M3NG9VP6QN0XK7J4CY")
	stepID := "generate-summary"

	t.Run("records a numeric run score", func(t *testing.T) {
		provider := &fakeScoreProvider{}
		service := NewService(ServiceOptions{Scores: provider})

		resp, err := service.CreateScore(context.Background(), &apiv2.CreateScoreRequest{
			RunId: runID.String(),
			Scores: []*apiv2.CreateScoreInput{
				{Name: "accuracy", Value: structpb.NewNumberValue(0.95)},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, provider.params)
		require.Equal(t, runID, provider.params.RunID)
		require.Len(t, provider.params.Scores, 1)
		require.Nil(t, provider.params.Scores[0].StepID)
		require.Equal(t, "accuracy", provider.params.Scores[0].Name)
		require.Equal(t, 0.95, provider.params.Scores[0].Value)
		require.Len(t, resp.Data, 1)
		require.Equal(t, runID.String(), resp.Data[0].RunId)
		require.Equal(t, "accuracy", resp.Data[0].Name)
		require.Equal(t, 0.95, resp.Data[0].Value.GetNumberValue())
		require.Nil(t, resp.Data[0].StepId)
		require.NotNil(t, resp.Metadata.FetchedAt)
	})

	t.Run("records a boolean step score", func(t *testing.T) {
		provider := &fakeScoreProvider{}
		service := NewService(ServiceOptions{Scores: provider})

		resp, err := service.CreateScore(context.Background(), &apiv2.CreateScoreRequest{
			RunId: runID.String(),
			Scores: []*apiv2.CreateScoreInput{
				{Name: "passed", Value: structpb.NewBoolValue(true), StepId: &stepID},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, provider.params)
		require.Len(t, provider.params.Scores, 1)
		require.Equal(t, &stepID, provider.params.Scores[0].StepID)
		require.Equal(t, true, provider.params.Scores[0].Value)
		require.Equal(t, &stepID, resp.Data[0].StepId)
		require.True(t, resp.Data[0].Value.GetBoolValue())
	})

	t.Run("records an experiment score", func(t *testing.T) {
		provider := &fakeScoreProvider{}
		service := NewService(ServiceOptions{Scores: provider})

		resp, err := service.CreateScore(context.Background(), &apiv2.CreateScoreRequest{
			RunId: runID.String(),
			Scores: []*apiv2.CreateScoreInput{
				{
					Name:  "accuracy",
					Value: structpb.NewNumberValue(0.98),
					Experiment: &apiv2.ScoreExperiment{
						Id:      "model-routing",
						Variant: "baseline",
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, provider.params)
		require.Len(t, provider.params.Scores, 1)
		require.NotNil(t, provider.params.Scores[0].Experiment)
		require.Equal(t, "model-routing", provider.params.Scores[0].Experiment.ExperimentName)
		require.Equal(t, "baseline", provider.params.Scores[0].Experiment.Variant)
		require.Len(t, resp.Data, 1)
		require.NotNil(t, resp.Data[0].Experiment)
		require.Equal(t, "model-routing", resp.Data[0].Experiment.Id)
		require.Equal(t, "baseline", resp.Data[0].Experiment.Variant)
	})

	t.Run("records multiple step scores", func(t *testing.T) {
		provider := &fakeScoreProvider{}
		service := NewService(ServiceOptions{Scores: provider})
		firstStepID := "extract"
		secondStepID := "summarize"

		resp, err := service.CreateScore(context.Background(), &apiv2.CreateScoreRequest{
			RunId: runID.String(),
			Scores: []*apiv2.CreateScoreInput{
				{Name: "accuracy", Value: structpb.NewNumberValue(0.95), StepId: &firstStepID},
				{Name: "passed", Value: structpb.NewBoolValue(true), StepId: &secondStepID},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, provider.params)
		require.Len(t, provider.params.Scores, 2)
		require.Equal(t, &firstStepID, provider.params.Scores[0].StepID)
		require.Equal(t, "accuracy", provider.params.Scores[0].Name)
		require.Equal(t, 0.95, provider.params.Scores[0].Value)
		require.Equal(t, &secondStepID, provider.params.Scores[1].StepID)
		require.Equal(t, "passed", provider.params.Scores[1].Name)
		require.Equal(t, true, provider.params.Scores[1].Value)
		require.Len(t, resp.Data, 2)
		require.Equal(t, &firstStepID, resp.Data[0].StepId)
		require.Equal(t, &secondStepID, resp.Data[1].StepId)
	})

	t.Run("not implemented without a provider", func(t *testing.T) {
		service := NewService(ServiceOptions{})

		resp, err := service.CreateScore(context.Background(), &apiv2.CreateScoreRequest{
			RunId: runID.String(),
			Scores: []*apiv2.CreateScoreInput{
				{Name: "accuracy", Value: structpb.NewNumberValue(1)},
			},
		})

		require.Nil(t, resp)
		require.ErrorContains(t, err, "not yet implemented")
	})

	invalid := []struct {
		name    string
		req     *apiv2.CreateScoreRequest
		message string
	}{
		{
			name: "missing run id",
			req: &apiv2.CreateScoreRequest{
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy", Value: structpb.NewNumberValue(1)},
				},
			},
			message: "Run ID is required",
		},
		{
			name:    "missing name",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Value: structpb.NewNumberValue(1)}}},
			message: "scores[0]: Score name is required",
		},
		{
			name:    "blank name",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Name: "   ", Value: structpb.NewNumberValue(1)}}},
			message: "scores[0]: Score name is required",
		},
		{
			name:    "name too long",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Name: strings.Repeat("a", metadata.MaxScoreNameByteLength+1), Value: structpb.NewNumberValue(1)}}},
			message: "scores[0]: Score name or value is invalid",
		},
		{
			name:    "missing value",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Name: "accuracy"}}},
			message: "scores[0]: Score value is required",
		},
		{
			name:    "invalid run id",
			req:     &apiv2.CreateScoreRequest{RunId: "not-a-ulid", Scores: []*apiv2.CreateScoreInput{{Name: "accuracy", Value: structpb.NewNumberValue(1)}}},
			message: "Run ID must be a valid ULID",
		},
		{
			name:    "string value",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Name: "accuracy", Value: structpb.NewStringValue("high")}}},
			message: "scores[0]: Score value must be a finite number or boolean",
		},
		{
			name:    "null value",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Name: "accuracy", Value: structpb.NewNullValue()}}},
			message: "scores[0]: Score value must be a finite number or boolean",
		},
		{
			name:    "name with single quote",
			req:     &apiv2.CreateScoreRequest{RunId: runID.String(), Scores: []*apiv2.CreateScoreInput{{Name: "bad'name", Value: structpb.NewNumberValue(1)}}},
			message: "scores[0]: Score name or value is invalid",
		},
		{
			name: "empty step id",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy", Value: structpb.NewNumberValue(1), StepId: func() *string { s := ""; return &s }()},
				},
			},
			message: "scores[0]: Step ID must not be empty",
		},
		{
			name: "blank step id",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy", Value: structpb.NewNumberValue(1), StepId: func() *string { s := "   "; return &s }()},
				},
			},
			message: "scores[0]: Step ID must not be empty",
		},
		{
			name: "experiment missing id",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy", Value: structpb.NewNumberValue(1), Experiment: &apiv2.ScoreExperiment{Variant: "baseline"}},
				},
			},
			message: "scores[0]: Experiment ID is required",
		},
		{
			name: "experiment missing variant",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy", Value: structpb.NewNumberValue(1), Experiment: &apiv2.ScoreExperiment{Id: "model-routing"}},
				},
			},
			message: "scores[0]: Experiment variant is required",
		},
		{
			name: "experiment id too long",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{
						Name:  "accuracy",
						Value: structpb.NewNumberValue(1),
						Experiment: &apiv2.ScoreExperiment{
							Id:      strings.Repeat("a", metadata.MaxScoreNameByteLength+1),
							Variant: "baseline",
						},
					},
				},
			},
			message: "scores[0]: Experiment ID must be at most 128 UTF-8 bytes",
		},
		{
			name: "experiment variant too long",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{
						Name:  "accuracy",
						Value: structpb.NewNumberValue(1),
						Experiment: &apiv2.ScoreExperiment{
							Id:      "model-routing",
							Variant: strings.Repeat("a", metadata.MaxScoreNameByteLength+1),
						},
					},
				},
			},
			message: "scores[0]: Experiment variant must be at most 128 UTF-8 bytes",
		},
		{
			name: "experiment with step id",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{
						Name:   "accuracy",
						Value:  structpb.NewNumberValue(1),
						StepId: &stepID,
						Experiment: &apiv2.ScoreExperiment{
							Id:      "model-routing",
							Variant: "baseline",
						},
					},
				},
			},
			message: "scores[0]: Experiment scores must be run-scoped",
		},
		{
			name: "empty scores",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
			},
			message: "At least one score is required",
		},
		{
			name: "too many scores",
			req: &apiv2.CreateScoreRequest{
				RunId:  runID.String(),
				Scores: testScoreInputs(maxScoreBatchSize + 1),
			},
			message: "At most 100 scores are allowed",
		},
		{
			name: "batch missing value",
			req: &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy"},
				},
			},
			message: "scores[0]: Score value is required",
		},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			provider := &fakeScoreProvider{}
			service := NewService(ServiceOptions{Scores: provider})

			resp, err := service.CreateScore(context.Background(), tc.req)

			require.Nil(t, resp)
			require.ErrorContains(t, err, tc.message)
			require.Nil(t, provider.params)
		})
	}

	providerErrors := []struct {
		name    string
		err     error
		message string
	}{
		{name: "scores not enabled", err: ErrScoresNotEnabled, message: "Scores are not enabled"},
		{name: "target not found", err: ErrScoreTargetNotFound, message: "Run or step not found"},
		{name: "span too large", err: metadata.ErrMetadataSpanTooLarge, message: "metadata size limit"},
		{name: "run size exceeded", err: metadata.ErrRunMetadataSizeExceeded, message: "metadata size limit"},
		{name: "internal error", err: errors.New("boom"), message: "Unable to record score"},
	}

	for _, tc := range providerErrors {
		t.Run(tc.name, func(t *testing.T) {
			provider := &fakeScoreProvider{err: tc.err}
			service := NewService(ServiceOptions{Scores: provider})

			resp, err := service.CreateScore(context.Background(), &apiv2.CreateScoreRequest{
				RunId: runID.String(),
				Scores: []*apiv2.CreateScoreInput{
					{Name: "accuracy", Value: structpb.NewNumberValue(1)},
				},
			})

			require.Nil(t, resp)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestScoreMetadataUpdate(t *testing.T) {
	update, err := ScoreMetadataUpdate("accuracy", 0.95)
	require.NoError(t, err)
	require.Equal(t, metadata.KindInngestScore, update.Kind())
	require.Equal(t, enums.MetadataOpcodeMerge, update.Op())
	require.Equal(t, metadata.Values{"accuracy": json.RawMessage(`{"value":0.95}`)}, update.RawUpdate.Values)

	update, err = ScoreMetadataUpdate("passed", true)
	require.NoError(t, err)
	require.Equal(t, metadata.KindInngestScore, update.Kind())
	require.Equal(t, metadata.Values{"passed": json.RawMessage(`{"value":true}`)}, update.RawUpdate.Values)

	_, err = ScoreMetadataUpdate("bad'name", 1.0)
	require.Error(t, err)

	_, err = ScoreMetadataUpdate(strings.Repeat("a", metadata.MaxScoreNameByteLength+1), 1.0)
	require.Error(t, err)
}

func TestScoreExperimentMetadataUpdate(t *testing.T) {
	update, err := ScoreExperimentMetadataUpdate(ScoreExperimentInput{
		ExperimentName: "model-routing",
		Variant:        "baseline",
	})

	require.NoError(t, err)
	require.Equal(t, metadata.KindInngestExperiment, update.Kind())
	require.Equal(t, enums.MetadataOpcodeMerge, update.Op())
	require.Equal(t, metadata.Values{
		"name":    json.RawMessage(`"model-routing"`),
		"variant": json.RawMessage(`"baseline"`),
	}, update.RawUpdate.Values)
}

func TestStateScoreProviderFlagDisabled(t *testing.T) {
	provider := NewStateScoreProvider(StateScoreProviderOptions{
		Auth: func(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
			return uuid.New(), uuid.New(), nil
		},
		Enabled: func(ctx context.Context, accountID uuid.UUID) bool { return false },
	})

	err := provider.CreateScores(context.Background(), CreateScoresParams{
		RunID: ulid.MustParse("01HP1ZX8M3NG9VP6QN0XK7J4CY"),
		Scores: []ScoreInput{
			{Name: "accuracy", Value: 1.0},
		},
	})
	require.ErrorIs(t, err, ErrScoresNotEnabled)
}

func TestStateScoreProviderUsesMetadataLoaderForFinalizedRun(t *testing.T) {
	runID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTQ")
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	appID := uuid.New()
	loaderCalled := false

	provider := NewStateScoreProvider(StateScoreProviderOptions{
		State: fakeScoreRunService{err: statev2.ErrMetadataNotFound},
		Auth: func(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
			return accountID, envID, nil
		},
		MissingStateLoader: func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error) {
			loaderCalled = true
			require.Equal(t, accountID, id.Tenant.AccountID)
			require.Equal(t, envID, id.Tenant.EnvID)
			require.Equal(t, runID, id.RunID)
			return &statev2.Metadata{
				ID: statev2.ID{
					RunID:      runID,
					FunctionID: functionID,
					Tenant: statev2.Tenant{
						AccountID: accountID,
						EnvID:     envID,
						AppID:     appID,
					},
				},
			}, nil
		},
	})

	err := provider.CreateScores(context.Background(), CreateScoresParams{
		RunID: runID,
		Scores: []ScoreInput{
			testScoreInput(t, ScoreInput{Name: "accuracy", Value: 1.0}),
		},
	})
	require.NoError(t, err)
	require.True(t, loaderCalled)
}

func TestStateScoreProviderRejectsMissingRun(t *testing.T) {
	runID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTQ")
	accountID := uuid.New()
	envID := uuid.New()
	loaderCalled := false

	provider := NewStateScoreProvider(StateScoreProviderOptions{
		State: fakeScoreRunService{err: statev2.ErrMetadataNotFound},
		Auth: func(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
			return accountID, envID, nil
		},
		MissingStateLoader: func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error) {
			loaderCalled = true
			return nil, statev2.ErrMetadataNotFound
		},
	})

	err := provider.CreateScores(context.Background(), CreateScoresParams{
		RunID: runID,
		Scores: []ScoreInput{
			testScoreInput(t, ScoreInput{Name: "accuracy", Value: 1.0}),
		},
	})

	require.ErrorIs(t, err, ErrScoreTargetNotFound)
	require.True(t, loaderCalled)
}

func TestStateScoreProviderForwardsMetadataSizeIncrement(t *testing.T) {
	runID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTQ")
	accountID := uuid.New()
	envID := uuid.New()
	state := &fakeScoreIncrementingRunService{
		metadata: statev2.Metadata{
			ID: statev2.ID{
				RunID: runID,
				Tenant: statev2.Tenant{
					AccountID: accountID,
					EnvID:     envID,
				},
			},
		},
	}

	provider := NewStateScoreProvider(StateScoreProviderOptions{
		State: state,
		Auth: func(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
			return accountID, envID, nil
		},
		MissingStateLoader: func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error) {
			require.Fail(t, "missing state loader should not run when state metadata loads")
			return nil, nil
		},
	})

	err := provider.CreateScores(context.Background(), CreateScoresParams{
		RunID: runID,
		Scores: []ScoreInput{
			testScoreInput(t, ScoreInput{Name: "accuracy", Value: 1.0}),
		},
	})

	require.NoError(t, err)
	require.Equal(t, runID, state.incrementID.RunID)
	require.Positive(t, state.incrementDelta)
}

func TestStateScoreProviderValidatesStepTargets(t *testing.T) {
	runID := ulid.MustParse("01KVBJWM98JHAJPC9K5EXVAQTQ")
	accountID := uuid.New()
	envID := uuid.New()
	existingStepID := "generate-summary"

	baseOpts := StateScoreProviderOptions{
		State: fakeScoreRunService{
			metadata: statev2.Metadata{
				ID: statev2.ID{
					RunID: runID,
					Tenant: statev2.Tenant{
						AccountID: accountID,
						EnvID:     envID,
					},
				},
				Stack: []string{scoreTraceStepID(existingStepID)},
			},
		},
		Auth: func(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
			return accountID, envID, nil
		},
	}

	t.Run("accepts SDK step IDs from live state", func(t *testing.T) {
		provider := NewStateScoreProvider(baseOpts)
		stepID := existingStepID

		err := provider.CreateScores(context.Background(), CreateScoresParams{
			RunID: runID,
			Scores: []ScoreInput{
				testScoreInput(t, ScoreInput{StepID: &stepID, Name: "accuracy", Value: 1.0}),
			},
		})

		require.NoError(t, err)
	})

	t.Run("rejects missing step IDs", func(t *testing.T) {
		provider := NewStateScoreProvider(baseOpts)
		stepID := "missing-step"

		err := provider.CreateScores(context.Background(), CreateScoresParams{
			RunID: runID,
			Scores: []ScoreInput{
				testScoreInput(t, ScoreInput{StepID: &stepID, Name: "accuracy", Value: 1.0}),
			},
		})

		require.ErrorIs(t, err, ErrScoreTargetNotFound)
	})

	t.Run("rejects already hashed step IDs", func(t *testing.T) {
		provider := NewStateScoreProvider(baseOpts)
		stepID := scoreTraceStepID(existingStepID)

		err := provider.CreateScores(context.Background(), CreateScoresParams{
			RunID: runID,
			Scores: []ScoreInput{
				testScoreInput(t, ScoreInput{StepID: &stepID, Name: "accuracy", Value: 1.0}),
			},
		})

		require.ErrorIs(t, err, ErrScoreTargetNotFound)
	})

	t.Run("preserves state load errors", func(t *testing.T) {
		loadErr := errors.New("state unavailable")
		opts := baseOpts
		opts.State = fakeScoreRunService{err: loadErr}
		provider := NewStateScoreProvider(opts)
		stepID := existingStepID

		err := provider.CreateScores(context.Background(), CreateScoresParams{
			RunID: runID,
			Scores: []ScoreInput{
				testScoreInput(t, ScoreInput{StepID: &stepID, Name: "accuracy", Value: 1.0}),
			},
		})

		require.ErrorIs(t, err, loadErr)
		require.NotErrorIs(t, err, ErrScoreTargetNotFound)
	})

	t.Run("falls back to trace validator", func(t *testing.T) {
		called := false
		opts := baseOpts
		opts.State = fakeScoreRunService{err: statev2.ErrMetadataNotFound}
		opts.StepValidator = func(ctx context.Context, params ScoreStepTargetValidatorParams) error {
			called = true
			require.Equal(t, runID, params.RunID)
			require.Equal(t, "generate-summary", params.StepID)
			require.Equal(t, scoreTraceStepID("generate-summary"), params.TraceStepID)
			return nil
		}
		opts.MissingStateLoader = func(ctx context.Context, id statev2.ID) (*statev2.Metadata, error) {
			return &statev2.Metadata{ID: id}, nil
		}
		provider := NewStateScoreProvider(opts)
		stepID := "generate-summary"

		err := provider.CreateScores(context.Background(), CreateScoresParams{
			RunID: runID,
			Scores: []ScoreInput{
				testScoreInput(t, ScoreInput{StepID: &stepID, Name: "accuracy", Value: 1.0}),
			},
		})

		require.NoError(t, err)
		require.True(t, called)
	})
}

func testScoreInput(t *testing.T, input ScoreInput) ScoreInput {
	t.Helper()

	updates := make([]metadata.Update, 0, 2)
	if input.Experiment != nil {
		update, err := ScoreExperimentMetadataUpdate(*input.Experiment)
		require.NoError(t, err)
		updates = append(updates, update)
	}

	update, err := ScoreMetadataUpdate(input.Name, input.Value)
	require.NoError(t, err)
	input.Metadata = append(updates, update)
	return input
}

func testScoreInputs(count int) []*apiv2.CreateScoreInput {
	scores := make([]*apiv2.CreateScoreInput, 0, count)
	for i := 0; i < count; i++ {
		scores = append(scores, &apiv2.CreateScoreInput{
			Name:  "accuracy",
			Value: structpb.NewNumberValue(1),
		})
	}
	return scores
}

type fakeScoreRunService struct {
	statev2.RunService
	metadata statev2.Metadata
	err      error
}

func (f fakeScoreRunService) LoadMetadata(ctx context.Context, id statev2.ID, _ ...statev2.LoadMetadataOption) (statev2.Metadata, error) {
	return f.metadata, f.err
}

type fakeScoreIncrementingRunService struct {
	statev2.RunService
	metadata       statev2.Metadata
	incrementID    statev2.ID
	incrementDelta int
}

func (f *fakeScoreIncrementingRunService) LoadMetadata(ctx context.Context, id statev2.ID, _ ...statev2.LoadMetadataOption) (statev2.Metadata, error) {
	return f.metadata, nil
}

func (f *fakeScoreIncrementingRunService) IncrementMetadataSize(ctx context.Context, id statev2.ID, delta int) error {
	f.incrementID = id
	f.incrementDelta += delta
	return nil
}
