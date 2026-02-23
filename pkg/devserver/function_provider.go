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
}

// NewFunctionProvider returns a FunctionProvider that looks up functions by
// slug or UUID using the given DevFunctionReader.
func NewFunctionProvider(reader cqrs.DevFunctionReader) apiv2.FunctionProvider {
	return &cqrsFunctionProvider{reader: reader}
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
			return inngest.DeployedFunction{
				ID:            fn.ID,
				Slug:          fn.Slug,
				AppID:         fn.AppID,
				AccountID:     consts.DevServerAccountID,
				EnvironmentID: consts.DevServerEnvID,
				Function:      *inngestFn,
			}, nil
		}
	}
	return inngest.DeployedFunction{}, fmt.Errorf("function not found: %s", identifier)
}
