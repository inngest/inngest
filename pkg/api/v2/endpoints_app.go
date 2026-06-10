package apiv2

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/enums"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) GetApp(ctx context.Context, req *apiv2.GetAppRequest) (*apiv2.GetAppResponse, error) {
	if req.AppId == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID is required")
	}

	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetApp_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no app was fetched.")
	}

	if s.apps == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get app is not yet implemented")
	}

	app, err := s.apps.GetApp(ctx, decodePathParam(req.AppId))
	if err != nil {
		if errors.Is(err, ErrAppNotFound) {
			return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "App not found")
		}
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch app")
	}

	return &apiv2.GetAppResponse{
		Data:     toApp(app),
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func toApp(app App) *apiv2.App {
	result := &apiv2.App{
		Id:            appPublicID(app),
		Name:          app.Name,
		Method:        toAppMethod(app.Method),
		IsArchived:    !app.ArchivedAt.IsZero() && app.ArchivedAt.Before(time.Now()),
		FunctionCount: int32(app.FunctionCount),
	}
	if !app.CreatedAt.IsZero() {
		result.CreatedAt = timestamppb.New(app.CreatedAt)
	}
	if !app.ArchivedAt.IsZero() {
		result.ArchivedAt = timestamppb.New(app.ArchivedAt)
	}
	result.AppVersion = nonEmptyString(app.AppVersion)
	result.LatestSync = toAppSync(app.LatestSync)
	return result
}

func toAppSync(sync *AppSync) *apiv2.AppSync {
	if sync == nil {
		return nil
	}

	result := &apiv2.AppSync{
		Status:      nonEmptyString(sync.Status),
		SdkLanguage: nonEmptyString(sync.SdkLanguage),
		SdkVersion:  nonEmptyString(sync.SdkVersion),
		Framework:   nonEmptyString(sync.Framework),
		Url:         nonEmptyString(sync.URL),
		Error:       nonEmptyString(sync.Error),
		AppVersion:  nonEmptyString(sync.AppVersion),
	}
	if !sync.StartedAt.IsZero() {
		result.StartedAt = timestamppb.New(sync.StartedAt)
	}
	if !sync.CompletedAt.IsZero() {
		result.CompletedAt = timestamppb.New(sync.CompletedAt)
	}
	return result
}

// appPublicID returns the app ID used in v2 API paths, falling back for older
// records that do not have a separate user-facing ID.
func appPublicID(app App) string {
	if app.ID != "" {
		return app.ID
	}
	if app.Name != "" {
		return app.Name
	}
	return app.InternalID.String()
}

func toAppMethod(method enums.AppMethod) apiv2.AppMethod {
	switch method {
	case enums.AppMethodConnect:
		return apiv2.AppMethod_APP_METHOD_CONNECT
	case enums.AppMethodAPI:
		return apiv2.AppMethod_APP_METHOD_API
	default:
		return apiv2.AppMethod_APP_METHOD_SERVE
	}
}

func nonEmptyString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
