// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// APIGatewayProxyRequest contains data coming from the API Gateway proxy
type APIGatewayProxyRequest struct {
	Resource                        string                        `json:"resource"` // The resource path defined in API Gateway
	Path                            string                        `json:"path"`     // The url path for the caller
	HTTPMethod                      string                        `json:"httpMethod"`
	Headers                         map[string]string             `json:"headers"`
	MultiValueHeaders               map[string][]string           `json:"multiValueHeaders"`
	QueryStringParameters           map[string]string             `json:"queryStringParameters"`
	MultiValueQueryStringParameters map[string][]string           `json:"multiValueQueryStringParameters"`
	PathParameters                  map[string]string             `json:"pathParameters"`
	StageVariables                  map[string]string             `json:"stageVariables"`
	RequestContext                  APIGatewayProxyRequestContext `json:"requestContext"`
	Body                            string                        `json:"body"`
	IsBase64Encoded                 bool                          `json:"isBase64Encoded,omitempty"`
}

// APIGatewayProxyResponse configures the response to be returned by API Gateway for the request
type APIGatewayProxyResponse struct {
	StatusCode        int                 `json:"statusCode"`
	Headers           map[string]string   `json:"headers"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
	Body              string              `json:"body"`
	IsBase64Encoded   bool                `json:"isBase64Encoded,omitempty"`
}

// APIGatewayProxyStreamingResponse configures the response to be returned by API Gateway for the request.
//   - integration type must be AWS_PROXY
//   - integration uri must be arn:<partition>:apigateway:<region>:lambda:path/2021-11-15/functions/<function-arn>/response-streaming-invocations
//   - integration response transfer mode must be STREAM
//
// If not using the above streaming integration, use APIGatewayProxyResponse instead
type APIGatewayProxyStreamingResponse struct {
	prelude *bytes.Buffer

	StatusCode        int
	Headers           map[string]string
	MultiValueHeaders map[string][]string
	Body              io.Reader
	Cookies           []string
}

func (r *APIGatewayProxyStreamingResponse) Read(p []byte) (n int, err error) {
	if r.prelude == nil {
		b, err := json.Marshal(struct {
			StatusCode        int                 `json:"statusCode,omitempty"`
			Headers           map[string]string   `json:"headers,omitempty"`
			MultiValueHeaders map[string][]string `json:"multiValueHeaders,omitempty"`
			Cookies           []string            `json:"cookies,omitempty"`
		}{
			StatusCode:        r.StatusCode,
			Headers:           r.Headers,
			MultiValueHeaders: r.MultiValueHeaders,
			Cookies:           r.Cookies,
		})
		if err != nil {
			return 0, err
		}
		r.prelude = bytes.NewBuffer(append(b, 0, 0, 0, 0, 0, 0, 0, 0))
	}
	if r.prelude.Len() > 0 {
		return r.prelude.Read(p)
	}
	if r.Body == nil {
		return 0, io.EOF
	}
	return r.Body.Read(p)
}

func (r *APIGatewayProxyStreamingResponse) Close() error {
	if closer, ok := r.Body.(io.ReadCloser); ok {
		return closer.Close()
	}
	return nil
}

func (r *APIGatewayProxyStreamingResponse) MarshalJSON() ([]byte, error) {
	return nil, errors.New("not json")
}

func (r *APIGatewayProxyStreamingResponse) ContentType() string {
	return "application/vnd.awslambda.http-integration-response"
}

// APIGatewayProxyRequestContext contains the information to identify the AWS account and resources invoking the
// Lambda function. It also includes Cognito identity information for the caller.
type APIGatewayProxyRequestContext struct {
	AccountID         string                    `json:"accountId"`
	ResourceID        string                    `json:"resourceId"`
	OperationName     string                    `json:"operationName,omitempty"`
	Stage             string                    `json:"stage"`
	DomainName        string                    `json:"domainName"`
	DomainPrefix      string                    `json:"domainPrefix"`
	RequestID         string                    `json:"requestId"`
	ExtendedRequestID string                    `json:"extendedRequestId"`
	Protocol          string                    `json:"protocol"`
	Identity          APIGatewayRequestIdentity `json:"identity"`
	ResourcePath      string                    `json:"resourcePath"`
	Path              string                    `json:"path"`
	Authorizer        map[string]interface{}    `json:"authorizer"`
	HTTPMethod        string                    `json:"httpMethod"`
	RequestTime       string                    `json:"requestTime"`
	RequestTimeEpoch  int64                     `json:"requestTimeEpoch"`
	APIID             string                    `json:"apiId"` // The API Gateway rest API Id
}

// APIGatewayV2HTTPRequest contains data coming from the new HTTP API Gateway
type APIGatewayV2HTTPRequest struct {
	Version               string                         `json:"version"`
	RouteKey              string                         `json:"routeKey"`
	RawPath               string                         `json:"rawPath"`
	RawQueryString        string                         `json:"rawQueryString"`
	Cookies               []string                       `json:"cookies,omitempty"`
	Headers               map[string]string              `json:"headers"`
	QueryStringParameters map[string]string              `json:"queryStringParameters,omitempty"`
	PathParameters        map[string]string              `json:"pathParameters,omitempty"`
	RequestContext        APIGatewayV2HTTPRequestContext `json:"requestContext"`
	StageVariables        map[string]string              `json:"stageVariables,omitempty"`
	Body                  string                         `json:"body,omitempty"`
	IsBase64Encoded       bool                           `json:"isBase64Encoded"`
}

// APIGatewayV2HTTPRequestContext contains the information to identify the AWS account and resources invoking the Lambda function.
type APIGatewayV2HTTPRequestContext struct {
	RouteKey       string                                               `json:"routeKey"`
	AccountID      string                                               `json:"accountId"`
	Stage          string                                               `json:"stage"`
	RequestID      string                                               `json:"requestId"`
	Authorizer     *APIGatewayV2HTTPRequestContextAuthorizerDescription `json:"authorizer,omitempty"`
	APIID          string                                               `json:"apiId"` // The API Gateway HTTP API Id
	DomainName     string                                               `json:"domainName"`
	DomainPrefix   string                                               `json:"domainPrefix"`
	Time           string                                               `json:"time"`
	TimeEpoch      int64                                                `json:"timeEpoch"`
	HTTP           APIGatewayV2HTTPRequestContextHTTPDescription        `json:"http"`
	Authentication APIGatewayV2HTTPRequestContextAuthentication         `json:"authentication,omitempty"`
}

// APIGatewayV2HTTPRequestContextAuthorizerDescription contains authorizer information for the request context.
type APIGatewayV2HTTPRequestContextAuthorizerDescription struct {
	JWT    *APIGatewayV2HTTPRequestContextAuthorizerJWTDescription `json:"jwt,omitempty"`
	Lambda map[string]interface{}                                  `json:"lambda,omitempty"`
	IAM    *APIGatewayV2HTTPRequestContextAuthorizerIAMDescription `json:"iam,omitempty"`
}

// APIGatewayV2HTTPRequestContextAuthorizerJWTDescription contains JWT authorizer information for the request context.
type APIGatewayV2HTTPRequestContextAuthorizerJWTDescription struct {
	Claims map[string]string `json:"claims"`
	Scopes []string          `json:"scopes,omitempty"`
}

// APIGatewayV2HTTPRequestContextAuthorizerIAMDescription contains IAM information for the request context.
type APIGatewayV2HTTPRequestContextAuthorizerIAMDescription struct {
	AccessKey       string                                                  `json:"accessKey"`
	AccountID       string                                                  `json:"accountId"`
	CallerID        string                                                  `json:"callerId"`
	CognitoIdentity APIGatewayV2HTTPRequestContextAuthorizerCognitoIdentity `json:"cognitoIdentity,omitempty"`
	PrincipalOrgID  string                                                  `json:"principalOrgId"`
	UserARN         string                                                  `json:"userArn"`
	UserID          string                                                  `json:"userId"`
}

// APIGatewayV2HTTPRequestContextAuthorizerCognitoIdentity contains Cognito identity information for the request context.
type APIGatewayV2HTTPRequestContextAuthorizerCognitoIdentity struct {
	AMR            []string `json:"amr"`
	IdentityID     string   `json:"identityId"`
	IdentityPoolID string   `json:"identityPoolId"`
}

// APIGatewayV2HTTPRequestContextHTTPDescription contains HTTP information for the request context.
type APIGatewayV2HTTPRequestContextHTTPDescription struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	Protocol  string `json:"protocol"`
	SourceIP  string `json:"sourceIp"`
	UserAgent string `json:"userAgent"`
}

// APIGatewayV2HTTPResponse configures the response to be returned by API Gateway V2 for the request
type APIGatewayV2HTTPResponse struct {
	StatusCode        int                 `json:"statusCode"`
	Headers           map[string]string   `json:"headers"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
	Body              string              `json:"body"`
	IsBase64Encoded   bool                `json:"isBase64Encoded,omitempty"`
	Cookies           []string            `json:"cookies"`
}

// APIGatewayRequestIdentity contains identity information for the request caller.
type APIGatewayRequestIdentity struct {
	CognitoIdentityPoolID         string                                                          `json:"cognitoIdentityPoolId,omitempty"`
	AccountID                     string                                                          `json:"accountId,omitempty"`
	CognitoIdentityID             string                                                          `json:"cognitoIdentityId,omitempty"`
	Caller                        string                                                          `json:"caller,omitempty"`
	APIKey                        string                                                          `json:"apiKey,omitempty"`
	APIKeyID                      string                                                          `json:"apiKeyId,omitempty"`
	AccessKey                     string                                                          `json:"accessKey,omitempty"`
	SourceIP                      string                                                          `json:"sourceIp"`
	CognitoAuthenticationType     string                                                          `json:"cognitoAuthenticationType,omitempty"`
	CognitoAuthenticationProvider string                                                          `json:"cognitoAuthenticationProvider,omitempty"`
	UserArn                       string                                                          `json:"userArn,omitempty"` //nolint: staticcheck
	UserAgent                     string                                                          `json:"userAgent"`
	User                          string                                                          `json:"user,omitempty"`
	ClientCert                    *APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCert `json:"clientCert,omitempty"`
}

// APIGatewayWebsocketProxyRequest contains data coming from the API Gateway proxy
type APIGatewayWebsocketProxyRequest struct {
	Resource                        string                                 `json:"resource,omitempty"` // The resource path defined in API Gateway
	Path                            string                                 `json:"path,omitempty"`     // The url path for the caller
	HTTPMethod                      string                                 `json:"httpMethod,omitempty"`
	Headers                         map[string]string                      `json:"headers,omitempty"`
	MultiValueHeaders               map[string][]string                    `json:"multiValueHeaders,omitempty"`
	QueryStringParameters           map[string]string                      `json:"queryStringParameters,omitempty"`
	MultiValueQueryStringParameters map[string][]string                    `json:"multiValueQueryStringParameters,omitempty"`
	PathParameters                  map[string]string                      `json:"pathParameters,omitempty"`
	StageVariables                  map[string]string                      `json:"stageVariables,omitempty"`
	RequestContext                  APIGatewayWebsocketProxyRequestContext `json:"requestContext"`
	Body                            string                                 `json:"body,omitempty"`
	IsBase64Encoded                 bool                                   `json:"isBase64Encoded"`
}

// APIGatewayWebsocketProxyRequestContext contains the information to identify
// the AWS account and resources invoking the Lambda function. It also includes
// Cognito identity information for the caller.
type APIGatewayWebsocketProxyRequestContext struct {
	AccountID            string                    `json:"accountId,omitempty"`
	ResourceID           string                    `json:"resourceId,omitempty"`
	Stage                string                    `json:"stage"`
	RequestID            string                    `json:"requestId"`
	Identity             APIGatewayRequestIdentity `json:"identity"`
	ResourcePath         string                    `json:"resourcePath,omitempty"`
	Authorizer           interface{}               `json:"authorizer,omitempty"`
	HTTPMethod           string                    `json:"httpMethod,omitempty"`
	APIID                string                    `json:"apiId"` // The API Gateway rest API Id
	ConnectedAt          int64                     `json:"connectedAt"`
	ConnectionID         string                    `json:"connectionId"`
	DomainName           string                    `json:"domainName"`
	Error                string                    `json:"error,omitempty"`
	EventType            string                    `json:"eventType"`
	ExtendedRequestID    string                    `json:"extendedRequestId"`
	IntegrationLatency   string                    `json:"integrationLatency,omitempty"`
	MessageDirection     string                    `json:"messageDirection"`
	MessageID            interface{}               `json:"messageId,omitempty"`
	RequestTime          string                    `json:"requestTime"`
	RequestTimeEpoch     int64                     `json:"requestTimeEpoch"`
	RouteKey             string                    `json:"routeKey"`
	Status               string                    `json:"status,omitempty"`
	DisconnectStatusCode int64                     `json:"disconnectStatusCode,omitempty"`
	DisconnectReason     *string                   `json:"disconnectReason,omitempty"`
}

// APIGatewayCustomAuthorizerRequestTypeRequestIdentity contains identity information for the request caller including certificate information if using mTLS.
type APIGatewayCustomAuthorizerRequestTypeRequestIdentity struct {
	APIKey     string                                                         `json:"apiKey"`
	SourceIP   string                                                         `json:"sourceIp"`
	ClientCert APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCert `json:"clientCert"`
}

// APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCert contains certificate information for the request caller if using mTLS.
type APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCert struct {
	ClientCertPem string                                                                 `json:"clientCertPem"`
	IssuerDN      string                                                                 `json:"issuerDN"`
	SerialNumber  string                                                                 `json:"serialNumber"`
	SubjectDN     string                                                                 `json:"subjectDN"`
	Validity      APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCertValidity `json:"validity"`
}

// APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCertValidity contains certificate validity information for the request caller if using mTLS.
type APIGatewayCustomAuthorizerRequestTypeRequestIdentityClientCertValidity struct {
	NotAfter  string `json:"notAfter"`
	NotBefore string `json:"notBefore"`
}

// APIGatewayV2HTTPRequestContextAuthentication contains authentication context information for the request caller including client certificate information if using mTLS.
type APIGatewayV2HTTPRequestContextAuthentication struct {
	ClientCert APIGatewayV2HTTPRequestContextAuthenticationClientCert `json:"clientCert"`
}

// APIGatewayV2HTTPRequestContextAuthenticationClientCert contains client certificate information for the request caller if using mTLS.
type APIGatewayV2HTTPRequestContextAuthenticationClientCert struct {
	ClientCertPem string                                                         `json:"clientCertPem"`
	IssuerDN      string                                                         `json:"issuerDN"`
	SerialNumber  string                                                         `json:"serialNumber"`
	SubjectDN     string                                                         `json:"subjectDN"`
	Validity      APIGatewayV2HTTPRequestContextAuthenticationClientCertValidity `json:"validity"`
}

// APIGatewayV2HTTPRequestContextAuthenticationClientCertValidity contains client certificate validity information for the request caller if using mTLS.
type APIGatewayV2HTTPRequestContextAuthenticationClientCertValidity struct {
	NotAfter  string `json:"notAfter"`
	NotBefore string `json:"notBefore"`
}

type APIGatewayV2CustomAuthorizerV1RequestTypeRequestContext struct {
	Path         string                                               `json:"path"`
	AccountID    string                                               `json:"accountId"`
	ResourceID   string                                               `json:"resourceId"`
	Stage        string                                               `json:"stage"`
	RequestID    string                                               `json:"requestId"`
	Identity     APIGatewayCustomAuthorizerRequestTypeRequestIdentity `json:"identity"`
	ResourcePath string                                               `json:"resourcePath"`
	HTTPMethod   string                                               `json:"httpMethod"`
	APIID        string                                               `json:"apiId"`
}

type APIGatewayV2CustomAuthorizerV1Request struct {
	Version               string                                                  `json:"version"`
	Type                  string                                                  `json:"type"`
	MethodArn             string                                                  `json:"methodArn"` //nolint: staticcheck
	IdentitySource        string                                                  `json:"identitySource"`
	AuthorizationToken    string                                                  `json:"authorizationToken"`
	Resource              string                                                  `json:"resource"`
	Path                  string                                                  `json:"path"`
	HTTPMethod            string                                                  `json:"httpMethod"`
	Headers               map[string]string                                       `json:"headers"`
	QueryStringParameters map[string]string                                       `json:"queryStringParameters"`
	PathParameters        map[string]string                                       `json:"pathParameters"`
	StageVariables        map[string]string                                       `json:"stageVariables"`
	RequestContext        APIGatewayV2CustomAuthorizerV1RequestTypeRequestContext `json:"requestContext"`
}

type APIGatewayV2CustomAuthorizerV2Request struct {
	Version               string                         `json:"version"`
	Type                  string                         `json:"type"`
	RouteArn              string                         `json:"routeArn"` //nolint: staticcheck
	IdentitySource        []string                       `json:"identitySource"`
	RouteKey              string                         `json:"routeKey"`
	RawPath               string                         `json:"rawPath"`
	RawQueryString        string                         `json:"rawQueryString"`
	Cookies               []string                       `json:"cookies"`
	Headers               map[string]string              `json:"headers"`
	QueryStringParameters map[string]string              `json:"queryStringParameters"`
	RequestContext        APIGatewayV2HTTPRequestContext `json:"requestContext"`
	PathParameters        map[string]string              `json:"pathParameters"`
	StageVariables        map[string]string              `json:"stageVariables"`
}

// APIGatewayCustomAuthorizerContext represents the expected format of an API Gateway custom authorizer response.
// Deprecated. Code should be updated to use the Authorizer map from APIGatewayRequestIdentity. Ex: Authorizer["principalId"]
type APIGatewayCustomAuthorizerContext struct {
	PrincipalID *string `json:"principalId"`
	StringKey   *string `json:"stringKey,omitempty"`
	NumKey      *int    `json:"numKey,omitempty"`
	BoolKey     *bool   `json:"boolKey,omitempty"`
}

// APIGatewayCustomAuthorizerRequestTypeRequestContext represents the expected format of an API Gateway custom authorizer response.
type APIGatewayCustomAuthorizerRequestTypeRequestContext struct {
	Path         string                                               `json:"path"`
	AccountID    string                                               `json:"accountId"`
	ResourceID   string                                               `json:"resourceId"`
	Stage        string                                               `json:"stage"`
	RequestID    string                                               `json:"requestId"`
	Identity     APIGatewayCustomAuthorizerRequestTypeRequestIdentity `json:"identity"`
	ResourcePath string                                               `json:"resourcePath"`
	HTTPMethod   string                                               `json:"httpMethod"`
	APIID        string                                               `json:"apiId"`
}

// APIGatewayCustomAuthorizerRequest contains data coming in to a custom API Gateway authorizer function.
type APIGatewayCustomAuthorizerRequest struct {
	Type               string `json:"type"`
	AuthorizationToken string `json:"authorizationToken"`
	MethodArn          string `json:"methodArn"` //nolint: staticcheck
}

// APIGatewayCustomAuthorizerRequestTypeRequest contains data coming in to a custom API Gateway authorizer function.
type APIGatewayCustomAuthorizerRequestTypeRequest struct {
	Type                            string                                              `json:"type"`
	MethodArn                       string                                              `json:"methodArn"` //nolint: staticcheck
	Resource                        string                                              `json:"resource"`
	Path                            string                                              `json:"path"`
	HTTPMethod                      string                                              `json:"httpMethod"`
	Headers                         map[string]string                                   `json:"headers"`
	MultiValueHeaders               map[string][]string                                 `json:"multiValueHeaders"`
	QueryStringParameters           map[string]string                                   `json:"queryStringParameters"`
	MultiValueQueryStringParameters map[string][]string                                 `json:"multiValueQueryStringParameters"`
	PathParameters                  map[string]string                                   `json:"pathParameters"`
	StageVariables                  map[string]string                                   `json:"stageVariables"`
	RequestContext                  APIGatewayCustomAuthorizerRequestTypeRequestContext `json:"requestContext"`
}

// APIGatewayCustomAuthorizerResponse represents the expected format of an API Gateway authorization response.
type APIGatewayCustomAuthorizerResponse struct {
	PrincipalID        string                           `json:"principalId"`
	PolicyDocument     APIGatewayCustomAuthorizerPolicy `json:"policyDocument"`
	Context            map[string]interface{}           `json:"context,omitempty"`
	UsageIdentifierKey string                           `json:"usageIdentifierKey,omitempty"`
}

// APIGatewayV2CustomAuthorizerSimpleResponse represents the simple format of an API Gateway V2 authorization response.
type APIGatewayV2CustomAuthorizerSimpleResponse struct {
	IsAuthorized bool                   `json:"isAuthorized"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// APIGatewayCustomAuthorizerPolicy represents an IAM policy.
//
// Note: This type exists for backwards compatibility.
// should reference IAMPolicyDocument directly instead.
type APIGatewayCustomAuthorizerPolicy IAMPolicyDocument

type APIGatewayV2CustomAuthorizerIAMPolicyResponse struct {
	PrincipalID    string                           `json:"principalId"`
	PolicyDocument APIGatewayCustomAuthorizerPolicy `json:"policyDocument"`
	Context        map[string]interface{}           `json:"context,omitempty"`
}
