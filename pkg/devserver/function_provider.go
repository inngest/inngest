package devserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
}

// NewFunctionProvider returns a FunctionProvider that looks up functions by
// app-scoped slug or UUID using the given FunctionReader.
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
		if publicDeployedFunctionID(appID, fn) == functionID {
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

func publicDeployedFunctionID(appID string, fn inngest.DeployedFunction) string {
	if fn.Function.Slug != "" && fn.Function.Slug != fn.Slug {
		return fn.Function.Slug
	}

	functionID := fn.Function.Slug
	if functionID == "" {
		functionID = fn.Slug
	}
	if appID != "" {
		return strings.TrimPrefix(functionID, appID+"-")
	}
	return functionID
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
