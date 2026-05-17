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

// functionLookupReader combines direct lookup methods needed by the function provider.
type functionLookupReader interface {
	GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error)
	GetFunctionByExternalID(ctx context.Context, wsID uuid.UUID, appID string, functionID string) (*cqrs.Function, error)
}

type cqrsFunctionProvider struct {
	reader functionLookupReader
	apps   cqrs.AppReader
}

// NewFunctionProvider returns a FunctionProvider that looks up functions by
// slug or UUID using direct database lookups.
func NewFunctionProvider(reader functionLookupReader) apiv2.FunctionProvider {
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
	fn, err := p.lookupFunction(ctx, identifier)
	if err != nil {
		return inngest.DeployedFunction{}, err
	}
	if fn == nil {
		return inngest.DeployedFunction{}, fmt.Errorf("function not found: %s", identifier)
	}

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

func (p *cqrsFunctionProvider) lookupFunction(ctx context.Context, identifier string) (*cqrs.Function, error) {
	id, err := uuid.Parse(identifier)
	if err == nil {
		fn, err := p.reader.GetFunctionByInternalUUID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("function not found by ID: %w", err)
		}
		return fn, nil
	}

	fn, err := p.reader.GetFunctionByExternalID(ctx, uuid.UUID{}, "", identifier)
	if err != nil {
		return nil, fmt.Errorf("function not found by slug: %w", err)
	}
	return fn, nil
}
