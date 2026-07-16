package devserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
)

type cqrsFunctionProvider struct {
	reader functionProviderReader
	apps   cqrs.AppReader
}

type functionProviderReader interface {
	cqrs.DevFunctionReader
	cqrs.FunctionReader
	cqrs.FunctionV2Reader
}

// NewFunctionProvider returns a FunctionProvider that looks up functions by
// app-scoped slug or UUID using the given function reader.
func NewFunctionProvider(reader functionProviderReader) apiv2.FunctionProvider {
	var apps cqrs.AppReader
	if appReader, ok := reader.(cqrs.AppReader); ok {
		apps = appReader
	}

	return &cqrsFunctionProvider{
		reader: reader,
		apps:   apps,
	}
}

func (p *cqrsFunctionProvider) GetFunction(ctx context.Context, identifier string) (inngest.DeployedFunction, error) {
	if fnID, err := uuid.Parse(identifier); err == nil {
		if fn, err := p.reader.GetFunctionByInternalUUID(ctx, fnID); err == nil {
			return p.toDeployedFunction(ctx, fn)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return inngest.DeployedFunction{}, err
		}
	}

	fns, err := p.reader.GetFunctions(ctx)
	if err != nil {
		return inngest.DeployedFunction{}, err
	}
	for _, fn := range fns {
		if fn.Slug == identifier || fn.ID.String() == identifier {
			return p.toDeployedFunction(ctx, fn)
		}
	}
	return inngest.DeployedFunction{}, fmt.Errorf("%w: %s", apiv2.ErrFunctionNotFound, identifier)
}

func (p *cqrsFunctionProvider) GetFunctionByApp(ctx context.Context, appID string, functionID string) (inngest.DeployedFunction, error) {
	fns, err := p.reader.GetFunctionsByAppExternalID(ctx, consts.DevServerEnvID, appID)
	if err != nil {
		return inngest.DeployedFunction{}, err
	}
	return p.findFunctionInApp(ctx, fns, appID, functionID)
}

func (p *cqrsFunctionProvider) GetFunctions(ctx context.Context, appID string, opts apiv2.GetFunctionsOpts) (*apiv2.GetFunctionsResult, error) {
	limit := opts.Limit
	if limit < 1 {
		limit = 1
	}

	fns, err := p.reader.GetFunctionsByApp(ctx, cqrs.GetFunctionsByAppOpts{
		WorkspaceID: consts.DevServerEnvID,
		AppName:     appID,
		Cursor:      opts.Cursor,
		Limit:       limit + 1,
	})
	if err != nil {
		return nil, err
	}

	result := make([]inngest.DeployedFunction, 0, limit)
	for _, fn := range fns {
		deployed, err := p.toDeployedFunctionWithAppName(fn, appID)
		if err != nil {
			return nil, err
		}
		result = append(result, deployed)
		if len(result) == limit+1 {
			break
		}
	}

	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}

	return &apiv2.GetFunctionsResult{
		Functions: result,
		HasMore:   hasMore,
	}, nil
}

func (p *cqrsFunctionProvider) findFunctionInApp(ctx context.Context, fns []*cqrs.Function, appID string, functionID string) (inngest.DeployedFunction, error) {
	deployedFns := make([]inngest.DeployedFunction, 0, len(fns))
	for _, fn := range fns {
		deployed, err := p.toDeployedFunction(ctx, fn)
		if err != nil {
			return inngest.DeployedFunction{}, err
		}
		deployedFns = append(deployedFns, deployed)
	}

	//
	// Prefer the app-scoped ID before accepting legacy combined IDs; users can
	// name a function with the app prefix, and that should still resolve exactly.
	for _, fn := range deployedFns {
		if apiv2.PublicFunctionID(appID, fn.Slug, fn.Function.Slug) == functionID {
			return fn, nil
		}
	}

	for _, fn := range deployedFns {
		if functionIDsMatch(fn, functionID, appID+"-"+functionID) {
			return fn, nil
		}
	}

	return inngest.DeployedFunction{}, fmt.Errorf("%w: %s/%s", apiv2.ErrFunctionNotFound, appID, functionID)
}

func functionIDsMatch(fn inngest.DeployedFunction, bareFunctionID string, prefixedFunctionID string) bool {
	return fn.Function.Slug == bareFunctionID ||
		fn.Function.Slug == prefixedFunctionID ||
		fn.Slug == bareFunctionID ||
		fn.Slug == prefixedFunctionID
}

func (p *cqrsFunctionProvider) toDeployedFunction(ctx context.Context, fn *cqrs.Function) (inngest.DeployedFunction, error) {
	inngestFn, err := fn.InngestFunction()
	if err != nil {
		return inngest.DeployedFunction{}, err
	}
	appName := ""
	if p.apps != nil {
		if app, err := p.apps.GetAppByID(ctx, fn.AppID); err == nil {
			appName = app.Name
		}
	}

	return inngest.DeployedFunction{
		ID:            fn.ID,
		Slug:          fn.Slug,
		AppID:         fn.AppID,
		AppName:       appName,
		AccountID:     consts.DevServerAccountID,
		EnvironmentID: consts.DevServerEnvID,
		Function:      *inngestFn,
	}, nil
}

func (p *cqrsFunctionProvider) toDeployedFunctionWithAppName(fn *cqrs.Function, appName string) (inngest.DeployedFunction, error) {
	inngestFn, err := fn.InngestFunction()
	if err != nil {
		return inngest.DeployedFunction{}, err
	}

	return inngest.DeployedFunction{
		ID:            fn.ID,
		Slug:          fn.Slug,
		AppID:         fn.AppID,
		AppName:       appName,
		AccountID:     consts.DevServerAccountID,
		EnvironmentID: consts.DevServerEnvID,
		Function:      *inngestFn,
		ArchivedAt:    fn.ArchivedAt,
	}, nil
}
