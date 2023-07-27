package awsgateway

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// NewTransformTripper creates a new transform tripper which attempts to
// automatically wrap and recreate HTTP requests if the external SDK is
// served via Lambda.
//
// NOTE: This is dev server only and should never be used in production.
// It has limitations regarding reading responses.
func NewTransformTripper(tripper http.RoundTripper) http.RoundTripper {
	if tripper == nil {
		tripper = http.DefaultTransport
	}
	return transformTripper{RoundTripper: tripper}
}

// transformTripper disobeys the standard library documentation by:
//
// - automatically upgrading to lambda-like gateway requests whenever
//   we see lambda-like error responses.
// - automatically parsing API gateway responses into regular responses.
type transformTripper struct {
	http.RoundTripper
}

func (t transformTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	isLambda := strings.Contains(req.URL.Path, "2015-03-31/functions/function/invocations")
	if isLambda {
		var err error
		req, err = TransformRequest(req)
		if err != nil {
			return nil, err
		}
	}

	resp, err := t.RoundTripper.RoundTrip(req)
	if !isLambda || err != nil {
		// Non-lambda or errored means just return what we have.
		return resp, err
	}

	return TransformResponse(resp)
}

func TransformResponse(resp *http.Response) (*http.Response, error) {
	// Unmarshal the response from the
	defer resp.Body.Close()

	body := events.APIGatewayProxyResponse{}
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return nil, err
	}

	resp.StatusCode = body.StatusCode

	if body.IsBase64Encoded {
		byt, _ := base64.StdEncoding.DecodeString(body.Body)
		resp.Body = io.NopCloser(bytes.NewReader(byt))
	} else {
		resp.Body = io.NopCloser(strings.NewReader(body.Body))
	}

	headers := http.Header{}
	for k, v := range body.Headers {
		headers[k] = []string{v}
	}
	for k, v := range body.MultiValueHeaders {
		headers[k] = v
	}
	resp.Header = headers

	return resp, err
}

// TransformRequest transforms a request to conform with the API gateway.
func TransformRequest(r *http.Request) (*http.Request, error) {
	queryParams := map[string]string{}
	for k, v := range r.URL.Query() {
		queryParams[k] = v[0]
	}

	var (
		body []byte
		err  error
	)

	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
	}

	t := events.APIGatewayProxyRequest{
		Path:       r.URL.Path,
		HTTPMethod: r.Method,
		Headers: map[string]string{
			"host":              r.URL.Host,
			"x-forwarded-proto": r.URL.Scheme,
		},
		QueryStringParameters: queryParams,
		MultiValueHeaders:     r.URL.Query(),
		Body:                  base64.RawStdEncoding.EncodeToString(body),
		IsBase64Encoded:       true,
	}

	buf := &bytes.Buffer{}
	if err = json.NewEncoder(buf).Encode(t); err != nil {
		return nil, err
	}

	return http.NewRequest(http.MethodPost, r.URL.String(), buf)
}
