package apiv2

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxScoreBatchSize                 = 100
	maxScoreExperimentFieldByteLength = metadata.MaxScoreNameByteLength
)

func (s *Service) CreateScore(ctx context.Context, req *apiv2.CreateScoreRequest) (*apiv2.CreateScoreResponse, error) {
	if req.RunId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "Run ID is required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_CreateScore_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited, "API rate limit exceeded. The request was rejected and no score was recorded.")
	}

	if s.scores == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Create score is not yet implemented")
	}

	runID, err := ulid.Parse(decodePathParam(req.RunId))
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Run ID must be a valid ULID")
	}

	scoreInputs, err := scoreInputsFromRequest(req)
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	err = s.scores.CreateScores(ctx, CreateScoresParams{
		RunID:  runID,
		Scores: scoreInputs,
	})
	switch {
	case errors.Is(err, ErrScoresNotEnabled):
		return nil, s.base.NewError(http.StatusForbidden, apiv2base.ErrorAccessDenied, "Scores are not enabled for this account")
	case errors.Is(err, metadata.ErrMetadataSpanTooLarge), errors.Is(err, metadata.ErrRunMetadataSizeExceeded):
		return nil, s.base.NewError(http.StatusRequestEntityTooLarge, apiv2base.ErrorValidationError, "Score exceeds the run metadata size limit")
	case err != nil:
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to record score")
	}

	scores := make([]*apiv2.Score, 0, len(scoreInputs))
	for _, score := range scoreInputs {
		var experiment *apiv2.ScoreExperiment
		if score.Experiment != nil {
			experiment = &apiv2.ScoreExperiment{
				Id:      score.Experiment.ExperimentName,
				Variant: score.Experiment.Variant,
			}
		}
		scores = append(scores, &apiv2.Score{
			RunId:      runID.String(),
			Name:       score.Name,
			Value:      scoreProtoValue(score.Value),
			StepId:     score.StepID,
			Experiment: experiment,
		})
	}

	return &apiv2.CreateScoreResponse{
		Data: scores,
		Metadata: &apiv2.ResponseMetadata{
			FetchedAt: timestamppb.Now(),
		},
	}, nil
}

func scoreInputsFromRequest(req *apiv2.CreateScoreRequest) ([]ScoreInput, error) {
	if len(req.Scores) == 0 {
		return nil, fmt.Errorf("At least one score is required")
	}
	if len(req.Scores) > maxScoreBatchSize {
		return nil, fmt.Errorf("At most %d scores are allowed", maxScoreBatchSize)
	}

	scores := make([]ScoreInput, 0, len(req.Scores))
	for i, score := range req.Scores {
		input, err := scoreInputFromFields(score.Name, score.Value, score.StepId, score.Experiment)
		if err != nil {
			return nil, fmt.Errorf("scores[%d]: %w", i, err)
		}
		scores = append(scores, input)
	}
	return scores, nil
}

func scoreInputFromFields(name string, rawValue *structpb.Value, stepID *string, experiment *apiv2.ScoreExperiment) (ScoreInput, error) {
	if strings.TrimSpace(name) == "" {
		return ScoreInput{}, fmt.Errorf("Score name is required")
	}
	if rawValue == nil {
		return ScoreInput{}, fmt.Errorf("Score value is required")
	}

	value, err := scoreValue(rawValue)
	if err != nil {
		return ScoreInput{}, err
	}

	if stepID != nil && strings.TrimSpace(*stepID) == "" {
		return ScoreInput{}, fmt.Errorf("Step ID must not be empty when provided")
	}
	if stepID != nil && experiment != nil {
		return ScoreInput{}, fmt.Errorf("Experiment scores must be run-scoped")
	}

	scoreUpdate, err := ScoreMetadataUpdate(name, value)
	if err != nil {
		return ScoreInput{}, fmt.Errorf("Score name or value is invalid; names must be at most %d UTF-8 bytes, must not contain control characters or single quotes, and values must be a finite number or boolean", metadata.MaxScoreNameByteLength)
	}

	updates := make([]metadata.Update, 0, 2)
	var experimentInput *ScoreExperimentInput
	if experiment != nil {
		if strings.TrimSpace(experiment.Id) == "" {
			return ScoreInput{}, fmt.Errorf("Experiment ID is required when experiment is provided")
		}
		if len(experiment.Id) > maxScoreExperimentFieldByteLength {
			return ScoreInput{}, fmt.Errorf("Experiment ID must be at most %d UTF-8 bytes", maxScoreExperimentFieldByteLength)
		}
		if strings.TrimSpace(experiment.Variant) == "" {
			return ScoreInput{}, fmt.Errorf("Experiment variant is required when experiment is provided")
		}
		if len(experiment.Variant) > maxScoreExperimentFieldByteLength {
			return ScoreInput{}, fmt.Errorf("Experiment variant must be at most %d UTF-8 bytes", maxScoreExperimentFieldByteLength)
		}
		experimentInput = &ScoreExperimentInput{
			ExperimentName: experiment.Id,
			Variant:        experiment.Variant,
		}
		experimentUpdate, err := ScoreExperimentMetadataUpdate(*experimentInput)
		if err != nil {
			return ScoreInput{}, fmt.Errorf("Experiment metadata is invalid")
		}
		updates = append(updates, experimentUpdate)
	}
	updates = append(updates, scoreUpdate)

	return ScoreInput{
		StepID:     stepID,
		Experiment: experimentInput,
		Name:       name,
		Value:      value,
		Metadata:   updates,
	}, nil
}

func scoreValue(value *structpb.Value) (any, error) {
	switch kind := value.GetKind().(type) {
	case *structpb.Value_NumberValue:
		if math.IsNaN(kind.NumberValue) || math.IsInf(kind.NumberValue, 0) {
			return nil, fmt.Errorf("Score value must be a finite number or boolean")
		}
		return kind.NumberValue, nil
	case *structpb.Value_BoolValue:
		return kind.BoolValue, nil
	default:
		return nil, fmt.Errorf("Score value must be a finite number or boolean")
	}
}

func scoreProtoValue(value any) *structpb.Value {
	switch v := value.(type) {
	case bool:
		return structpb.NewBoolValue(v)
	case float64:
		return structpb.NewNumberValue(v)
	default:
		return structpb.NewNullValue()
	}
}
