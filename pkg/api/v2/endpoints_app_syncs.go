package apiv2

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/logger"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) ListAppSyncs(ctx context.Context, req *apiv2.ListAppSyncsRequest) (*apiv2.ListAppSyncsResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_ListAppSyncs_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited, "API rate limit exceeded. No syncs were fetched.")
	}
	if req.GetAppId() == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID is required")
	}
	if s.appSyncs == nil || s.apps == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "List app syncs is not yet implemented")
	}
	cursor, limit, err := appsPageOpts(req.GetCursor(), req.GetLimit())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, err.Error())
	}
	app, err := s.apps.GetApp(ctx, decodePathParam(req.GetAppId()))
	if errors.Is(err, ErrAppNotFound) {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "App not found")
	}
	if err != nil {
		logger.From(ctx).Error("unable to fetch app", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch app")
	}
	result, err := s.appSyncs.ListAppSyncs(ctx, app, ListAppSyncsOpts{Cursor: cursor, Limit: limit})
	if errors.Is(err, ErrInvalidCursor) {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Cursor is invalid")
	}
	if err != nil {
		logger.From(ctx).Error("unable to fetch app syncs", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch app syncs")
	}
	data := make([]*apiv2.Sync, 0, len(result.Syncs))
	for _, sync := range result.Syncs {
		data = append(data, toSync(app, sync))
	}
	page := &apiv2.Page{Limit: int32(limit), HasMore: result.HasMore}
	if result.HasMore && len(result.Syncs) > 0 {
		page.Cursor = new(result.Syncs[len(result.Syncs)-1].ID.String())
	}
	return &apiv2.ListAppSyncsResponse{
		Data:     data,
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
		Page:     page,
	}, nil
}

func (s *Service) GetAppSync(ctx context.Context, req *apiv2.GetAppSyncRequest) (*apiv2.GetAppSyncResponse, error) {
	if result := s.rateLimiter.CheckRateLimit(ctx, apiv2.V2_GetAppSync_FullMethodName); result.Limited {
		return nil, s.base.NewError(http.StatusTooManyRequests, apiv2base.ErrorRateLimited, "API rate limit exceeded. No sync was fetched.")
	}
	if req.GetAppId() == "" {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorMissingField, "App ID is required")
	}
	if s.appSyncs == nil || s.apps == nil {
		return nil, s.base.NewError(http.StatusNotImplemented, apiv2base.ErrorNotImplemented, "Get app sync is not yet implemented")
	}
	id, err := uuid.Parse(req.GetSyncId())
	if err != nil {
		return nil, s.base.NewError(http.StatusBadRequest, apiv2base.ErrorInvalidFieldFormat, "Sync ID must be a valid UUID")
	}
	app, err := s.apps.GetApp(ctx, decodePathParam(req.GetAppId()))
	if errors.Is(err, ErrAppNotFound) {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "App not found")
	}
	if err != nil {
		logger.From(ctx).Error("unable to fetch app", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch app")
	}
	sync, err := s.appSyncs.GetAppSync(ctx, app, id)
	if errors.Is(err, ErrAppSyncNotFound) {
		return nil, s.base.NewError(http.StatusNotFound, apiv2base.ErrorNotFound, "Sync not found")
	}
	if err != nil {
		logger.From(ctx).Error("unable to fetch app sync", "error", err)
		return nil, s.base.NewError(http.StatusInternalServerError, apiv2base.ErrorInternalError, "Unable to fetch app sync")
	}
	return &apiv2.GetAppSyncResponse{
		Data:     toSync(app, sync),
		Metadata: &apiv2.ResponseMetadata{FetchedAt: timestamppb.Now()},
	}, nil
}

func toSync(app App, sync AppSyncRecord) *apiv2.Sync {
	result := &apiv2.Sync{
		Id:          sync.ID.String(),
		AppId:       appPublicID(app),
		Status:      sync.Status,
		Url:         nonEmptyString(sync.URL),
		SdkLanguage: nonEmptyString(sync.SDKLanguage),
		SdkVersion:  nonEmptyString(sync.SDKVersion),
		Framework:   nonEmptyString(sync.Framework),
		AppVersion:  nonEmptyString(sync.AppVersion),
		CreatedAt:   timestamppb.New(sync.CreatedAt),
	}
	if sync.Error != nil {
		result.Error = &apiv2.SyncAppError{Code: sync.Error.Code, Message: sync.Error.Message}
	}
	return result
}
