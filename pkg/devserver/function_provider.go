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

type functionIDReader interface {
	GetFunctionsByInternalUUIDs(ctx context.Context, fnIDs []uuid.UUID) ([]*cqrs.Function, error)
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
			return p.deployedFunction(ctx, fn)
		}
	}
	return inngest.DeployedFunction{}, fmt.Errorf("function not found: %s", identifier)
}

func (p *cqrsFunctionProvider) GetFunctions(ctx context.Context, identifiers []string) (map[string]inngest.DeployedFunction, error) {
	requested := make(map[string]struct{}, len(identifiers))
	ids := make([]uuid.UUID, 0, len(identifiers))
	hasSlug := false
	for _, identifier := range identifiers {
		requested[identifier] = struct{}{}
		if id, err := uuid.Parse(identifier); err == nil {
			ids = append(ids, id)
		} else {
			hasSlug = true
		}
	}

	var fns []*cqrs.Function
	if idReader, ok := p.reader.(functionIDReader); ok && !hasSlug {
		var err error
		fns, err = idReader.GetFunctionsByInternalUUIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	} else {
		fetched, err := p.reader.GetFunctions(ctx)
		if err != nil {
			return nil, err
		}
		fns = fetched
	}

	result := make(map[string]inngest.DeployedFunction, len(identifiers))
	for _, fn := range fns {
		if _, ok := requested[fn.Slug]; !ok {
			if _, ok := requested[fn.ID.String()]; !ok {
				continue
			}
		}

		deployedFn, err := p.deployedFunction(ctx, fn)
		if err != nil {
			return nil, err
		}
		if _, ok := requested[fn.Slug]; ok {
			result[fn.Slug] = deployedFn
		}
		if _, ok := requested[fn.ID.String()]; ok {
			result[fn.ID.String()] = deployedFn
		}
	}

	return result, nil
}

func (p *cqrsFunctionProvider) deployedFunction(ctx context.Context, fn *cqrs.Function) (inngest.DeployedFunction, error) {
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
