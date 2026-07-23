package apiv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultAppsLimit = 20
	maxAppsLimit     = 100
)

func (s *Service) GetApps(ctx context.Context, req *apiv2.GetAppsRequest) (*apiv2.GetAppsResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetApps_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited,
			"API rate limit exceeded. The request was rejected and no apps were fetched.")
	}

	if s.apps == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get apps is not yet implemented")
	}

	cursor, limit, err := appsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}

	result, err := s.apps.GetApps(ctx, GetAppsOpts{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		logger.From(ctx).Error("unable to fetch apps", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch apps")
	}
	if result == nil {
		result = &GetAppsResult{}
	}

	data := make([]*apiv2.App, 0, len(result.Apps))
	for _, app := range result.Apps {
		data = append(data, toApp(app))
	}

	return &apiv2.GetAppsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     appsPage(result.Apps, limit, result.HasMore),
	}, nil
}

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

func appsPageOpts(cursor string, requestedLimit int32) (uuid.UUID, int, error) {
	limit := int(requestedLimit)
	if limit == 0 {
		limit = defaultAppsLimit
	}
	if limit < 1 {
		return uuid.Nil, 0, fmt.Errorf("Limit must be at least 1")
	}
	if limit > maxAppsLimit {
		return uuid.Nil, 0, fmt.Errorf("Limit cannot exceed %d", maxAppsLimit)
	}

	parsedCursor := uuid.Nil
	if cursor != "" {
		decodedCursor, err := uuid.Parse(cursor)
		if err != nil {
			return uuid.Nil, 0, fmt.Errorf("Cursor is invalid")
		}
		parsedCursor = decodedCursor
	}

	return parsedCursor, limit, nil
}

func appsPage(apps []App, limit int, hasMore bool) *apiv2.Page {
	page := &apiv2.Page{
		HasMore: hasMore,
		Limit:   int32(limit),
	}
	if hasMore && len(apps) > 0 {
		nextCursor := apps[len(apps)-1].InternalID.String()
		page.Cursor = &nextCursor
	}
	return page
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
	if !sync.SyncedAt.IsZero() {
		result.SyncedAt = timestamppb.New(sync.SyncedAt)
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
