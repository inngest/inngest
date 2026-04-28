package sdk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/inngest/inngest/pkg/util"
)

var (
	ErrNoFunctions = fmt.Errorf("No functions registered within your app")
)

type DeployType string

const (
	DeployTypePing    DeployType = "ping"
	DeployTypeConnect DeployType = "connect"
)

type FromReadCloserOpts struct {
	Env        string
	ForceHTTPS bool
	Platform   string
}

// FromReadCloserOpts creates a new RegisterRequest from an io.ReadCloser (e.g.
// an HTTP request body). It will also normalize the RegisterRequest.
func FromReadCloser(r io.ReadCloser, opts FromReadCloserOpts) (RegisterRequest, error) {
	fr := RegisterRequest{}
	err := json.NewDecoder(r).Decode(&fr)
	if err != nil {
		return fr, err
	}

	fr.Headers.Env = opts.Env
	fr.Headers.Platform = opts.Platform

	err = fr.normalize(opts.ForceHTTPS)
	if err != nil {
		return fr, err
	}

	return fr, nil
}

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
	DeployType DeployType `json:"deployType"`
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
	// AppVersion represents an optional application version identifier. This should change
	// whenever code within one of your Inngest function or any dependency thereof changes.
	AppVersion string `json:"appVersion,omitempty"`
	// Functions represents all functions hosted within this deploy.
	Functions []SDKFunction `json:"functions"`
	// Headers are fetched from the incoming HTTP request.  They are present
	// on all calls to Inngest from the SDK, and are separate from the RegisterRequest
	// JSON payload to have a single source of truth.
	Headers Headers `json:"headers"`

	// IdempotencyKey is an optional input to deduplicate syncs.
	IdempotencyKey string `json:"idempotencyKey"`

	// checksum is a memoized field.
	checksum string

	Capabilities Capabilities `json:"capabilities"`
}

const (
	InBandSyncV1 string = "v1"
	TrustProbeV1 string = "v1"
	ConnectV1    string = "v1"
)

type Capabilities struct {
	InBandSync string `json:"in_band_sync"`
	TrustProbe string `json:"trust_probe"`
	Connect    string `json:"connect"`
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

func (f RegisterRequest) IsConnect() bool {
	return f.Capabilities.Connect == ConnectV1 && f.DeployType == DeployTypeConnect
}


func (f *RegisterRequest) normalize(forceHTTPS bool) error {
	f.URL = util.NormalizeAppURL(f.URL, forceHTTPS)

	for _, fn := range f.Functions {
		for _, step := range fn.Steps {
			if rawStepURL, ok := step.Runtime["url"]; ok {
				if stepURL, ok := rawStepURL.(string); ok {
					step.Runtime["url"] = util.NormalizeAppURL(stepURL, forceHTTPS)
				}
			}
		}
	}

	return nil
}
