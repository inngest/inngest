package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
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
	Functions []SDKFunction `json:"functions"`
	// Headers are fetched from the incoming HTTP request.  They are present
	// on all calls to Inngest from the SDK, and are separate from the RegisterRequest
	// JSON payload to have a single source of truth.
	Headers Headers `json:"headers"`

	// checksum is a memoized field.
	checksum string
}

type Headers struct {
	Env      string `json:"env"`
	Platform string `json:"platform"`
}

func (f *RegisterRequest) Checksum() (string, error) {
	if f.checksum != "" {
		return f.checksum, nil
	}
	byt, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(byt)
	f.checksum = hex.EncodeToString(sum[:])
	return f.checksum, nil
}

func (f RegisterRequest) SDKLanguage() string {
	parts := strings.Split(f.SDK, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (f RegisterRequest) SDKVersion() string {
	parts := strings.Split(f.SDK, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Parse parses the incoming
func (f RegisterRequest) Parse(ctx context.Context) ([]*inngest.Function, error) {
	// Ensure that there are no functions with the same ID.
	if len(f.Functions) == 0 {
		return nil, ErrNoFunctions
	}

	// err is a multierror which stores all function and validation errors for easy
	// reporting and debugging.
	var err error

	funcs := make([]*inngest.Function, len(f.Functions))

	for n, sdkFn := range f.Functions {
		var ferr error
		if len(sdkFn.Steps) == 0 {
			err = multierror.Append(err, fmt.Errorf("Function has no steps: %s", sdkFn.Name))
			continue
		}

		fn, ferr := sdkFn.Function()
		if ferr != nil {
			err = multierror.Append(err, ferr)
			continue
		}
		funcs[n] = fn

		if ferr := fn.Validate(ctx); ferr != nil {
			err = multierror.Append(err, ferr)
		}

		for n, step := range fn.Steps {
			uri, ferr := url.Parse(step.URI)
			if ferr != nil {
				err = multierror.Append(err, fmt.Errorf("Step '%s' has an invalid URI", step.ID))
			}
			if uri.Scheme != "http" && uri.Scheme != "https" {
				err = multierror.Append(err, fmt.Errorf("Step '%s' has an invalid driver. Only HTTP drivers may be used with SDK functions.", step.ID))
				continue
			}
			step.URI = util.NormalizeAppURL(step.URI)
			fn.Steps[n] = step
		}
	}

	return funcs, err
}
