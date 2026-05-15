package devserver

import (
	"context"
	"fmt"

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
	fns, err := p.reader.GetFunctions(ctx)
	if err != nil {
		return inngest.DeployedFunction{}, err
	}
	for _, fn := range fns {
		if fn.Slug == identifier || fn.ID.String() == identifier {
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
	}
	return inngest.DeployedFunction{}, fmt.Errorf("function not found: %s", identifier)
}
