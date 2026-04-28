package events

type S3ObjectLambdaEvent struct {
	XAmzRequestID        string                              `json:"xAmzRequestId"`
	GetObjectContext     *S3ObjectLambdaGetObjectContext     `json:"getObjectContext,omitempty"`
	ListObjectsContext   *S3ObjectLambdaListObjectsContext   `json:"listObjectsContext,omitempty"`
	ListObjectsV2Context *S3ObjectLambdaListObjectsV2Context `json:"listObjectsV2Context,omitempty"`
	HeadObjectContext    *S3ObjectLambdaHeadObjectContext    `json:"headObjectContext,omitempty"`
	Configuration        S3ObjectLambdaConfiguration         `json:"configuration"`
	UserRequest          S3ObjectLambdaUserRequest           `json:"userRequest"`
	UserIdentity         S3ObjectLambdaUserIdentity          `json:"userIdentity"`
	ProtocolVersion      string                              `json:"protocolVersion"`
}

type S3ObjectLambdaGetObjectContext struct {
	InputS3URL  string `json:"inputS3Url"`
	OutputRoute string `json:"outputRoute"`
	OutputToken string `json:"outputToken"`
}

type S3ObjectLambdaListObjectsContext struct {
	InputS3URL string `json:"inputS3Url"`
}

type S3ObjectLambdaListObjectsV2Context struct {
	InputS3URL string `json:"inputS3Url"`
}

type S3ObjectLambdaHeadObjectContext struct {
	InputS3URL string `json:"inputS3Url"`
}

type S3ObjectLambdaConfiguration struct {
	AccessPointARN           string `json:"accessPointArn"`
	SupportingAccessPointARN string `json:"supportingAccessPointArn"`
	Payload                  string `json:"payload"`
}

type S3ObjectLambdaUserRequest struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type S3ObjectLambdaUserIdentity struct {
	Type           string                        `json:"type"`
	PrincipalID    string                        `json:"principalId"`
	ARN            string                        `json:"arn"`
	AccountID      string                        `json:"accountId"`
	AccessKeyID    string                        `json:"accessKeyId"`
	SessionContext *S3ObjectLambdaSessionContext `json:"sessionContext,omitempty"`
}

type S3ObjectLambdaSessionContext struct {
	Attributes    map[string]string            `json:"attributes"`
	SessionIssuer *S3ObjectLambdaSessionIssuer `json:"sessionIssuer,omitempty"`
}

type S3ObjectLambdaSessionIssuer struct {
	Type        string `json:"type"`
	PrincipalID string `json:"principalId"`
	ARN         string `json:"arn"`
	AccountID   string `json:"accountId"`
	UserName    string `json:"userName"`
}
