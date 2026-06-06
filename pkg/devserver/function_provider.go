package devserver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/inngest"
)

type cqrsFunctionProvider struct {
	reader cqrs.DevFunctionReader
	apps   cqrs.AppReader
}

// NewFunctionProvider returns a FunctionProvider that looks up functions by
// slug or UUID using the given DevFunctionReader.
func NewFunctionProvider(reader cqrs.DevFunctionReader) apiv2.FunctionProvider {
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
		if reader, ok := p.reader.(cqrs.FunctionReader); ok {
			if fn, err := reader.GetFunctionByInternalUUID(ctx, fnID); err == nil {
				return p.toDeployedFunction(ctx, fn)
			}
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
	return inngest.DeployedFunction{}, fmt.Errorf("function not found: %s", identifier)
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
