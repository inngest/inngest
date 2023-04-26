package sdk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/inngest"
)

var (
	ErrNoFunctions = fmt.Errorf("No functions registered within your app")
)

// RegisterRequest represents a new deploy request from SDK-based functions.
// This lets us know that a new deploy has started and that we need to
// upsert function definitions from the newly deployed endpoint.
type RegisterRequest struct {
	// V is the version for this response, which lets us upgrade the SDKs
	// and APIs with backwards compatiblity.
	V string `json:"v"`
	// URL represents the entire URL which hosts the functions, eg.
	// https://www.example.com/api/v1/inngest
	URL string `json:"url"`
	// DeployType represents how this was deployed, eg. via a ping.
	// This allows us to change flows in the future, or support
	// multiple registration flows within a single fetch response.
	DeployType string `json:"deployType"`
	// SDK represents the SDK language and version used for these
	// functions, in the format: "js:v0.1.0"
	SDK string `json:"sdk"`
	// Framework represents the framework used to host these functions.
	// For example, using the JS SDK we support NextJS, Netlify, Express,
	// etc via middleware to initialize the SDK handler.  This lets us
	// gather stats on usage.
	Framework string `json:"framework"`
	// AppName represents a namespaced app name for each deployed function.
	AppName string `json:"appName"`
	// Functions represents all functions hosted within this deploy.
	Functions []inngest.Function `json:"functions"` // TODO: This is a custom datatype
	// Hash represents the commit checksum for the deploy.
	Hash *string `json:"hash"`
}

func (f RegisterRequest) Validate(ctx context.Context) error {
	// Ensure that there are no functions with the same ID.
	if len(f.Functions) == 0 {
		return ErrNoFunctions
	}

	var err error

	for _, fn := range f.Functions {
		if len(fn.Steps) == 0 {
			return fmt.Errorf("Function has no steps: %s", fn.Name)
		}

		for _, step := range fn.Steps {
			if verr := fn.Validate(ctx); verr != nil {
				err = multierror.Append(err, verr)
			}

			uri, perr := url.Parse(step.URI)
			if perr != nil {
				err = multierror.Append(err, fmt.Errorf("Step '%s' has an invalid URI", step.ID))
			}
			if uri.Scheme != "http" && uri.Scheme != "https" {
				err = multierror.Append(err, fmt.Errorf("Step '%s' has an invalid driver. Only HTTP drivers may be used with SDK functions.", step.ID))
				continue
			}
		}
	}

	return err
}
