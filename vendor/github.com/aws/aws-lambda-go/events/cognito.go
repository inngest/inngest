// Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package events

// CognitoEvent contains data from an event sent from Amazon Cognito Sync
type CognitoEvent struct {
	DatasetName    string                          `json:"datasetName"`
	DatasetRecords map[string]CognitoDatasetRecord `json:"datasetRecords"`
	EventType      string                          `json:"eventType"`
	IdentityID     string                          `json:"identityId"`
	IdentityPoolID string                          `json:"identityPoolId"`
	Region         string                          `json:"region"`
	Version        int                             `json:"version"`
}

// CognitoDatasetRecord represents a record from an Amazon Cognito Sync event
type CognitoDatasetRecord struct {
	NewValue string `json:"newValue"`
	OldValue string `json:"oldValue"`
	Op       string `json:"op"`
}

// CognitoEventUserPoolsPreSignup is sent by Amazon Cognito User Pools when a user attempts to register
// (sign up), allowing a Lambda to perform custom validation to accept or deny the registration request
type CognitoEventUserPoolsPreSignup struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPreSignupRequest  `json:"request"`
	Response CognitoEventUserPoolsPreSignupResponse `json:"response"`
}

// CognitoEventUserPoolsPreAuthentication is sent by Amazon Cognito User Pools when a user submits their information
// to be authenticated, allowing you to perform custom validations to accept or deny the sign in request.
type CognitoEventUserPoolsPreAuthentication struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPreAuthenticationRequest  `json:"request"`
	Response CognitoEventUserPoolsPreAuthenticationResponse `json:"response"`
}

// CognitoEventUserPoolsPostConfirmation is sent by Amazon Cognito User Pools after a user is confirmed,
// allowing the Lambda to send custom messages or add custom logic.
type CognitoEventUserPoolsPostConfirmation struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPostConfirmationRequest  `json:"request"`
	Response CognitoEventUserPoolsPostConfirmationResponse `json:"response"`
}

// CognitoEventUserPoolsPreTokenGen is sent by Amazon Cognito User Pools when a user attempts to retrieve
// credentials, allowing a Lambda to perform insert, suppress or override claims
type CognitoEventUserPoolsPreTokenGen struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPreTokenGenRequest  `json:"request"`
	Response CognitoEventUserPoolsPreTokenGenResponse `json:"response"`
}

// CognitoEventUserPoolsPreTokenGenV2 is sent by Amazon Cognito User Pools when a user attempts to retrieve
// credentials, allowing a Lambda to perform insert, suppress or override claims and scopes
//
// Deprecated: Use CognitoEventUserPoolsPreTokenGenV2_0 instead.
// This struct incorrectly restricts the ClaimsToAddOrOverride values as strings, but Cogntio supports any type.
type CognitoEventUserPoolsPreTokenGenV2 struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPreTokenGenV2Request  `json:"request"`
	Response CognitoEventUserPoolsPreTokenGenV2Response `json:"response"`
}

// CognitoEventUserPoolsPreTokenGenV2_0 is sent by Amazon Cognito User Pools when a user attempts to retrieve
// credentials, allowing a Lambda to perform insert, suppress or override claims and scopes
type CognitoEventUserPoolsPreTokenGenV2_0 struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPreTokenGenRequestV2_0  `json:"request"`
	Response CognitoEventUserPoolsPreTokenGenResponseV2_0 `json:"response"`
}

// CognitoEventUserPoolsPostAuthentication is sent by Amazon Cognito User Pools after a user is authenticated,
// allowing the Lambda to add custom logic.
type CognitoEventUserPoolsPostAuthentication struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPostAuthenticationRequest  `json:"request"`
	Response CognitoEventUserPoolsPostAuthenticationResponse `json:"response"`
}

// CognitoEventUserPoolsMigrateUser is sent by Amazon Cognito User Pools when a user does not exist in the
// user pool at the time of sign-in with a password, or in the forgot-password flow.
type CognitoEventUserPoolsMigrateUser struct {
	CognitoEventUserPoolsHeader
	CognitoEventUserPoolsMigrateUserRequest  `json:"request"`
	CognitoEventUserPoolsMigrateUserResponse `json:"response"`
}

// CognitoEventUserPoolsCallerContext contains information about the caller
type CognitoEventUserPoolsCallerContext struct {
	AWSSDKVersion string `json:"awsSdkVersion"`
	ClientID      string `json:"clientId"`
}

// CognitoEventUserPoolsHeader contains common data from events sent by Amazon Cognito User Pools
type CognitoEventUserPoolsHeader struct {
	Version       string                             `json:"version"`
	TriggerSource string                             `json:"triggerSource"`
	Region        string                             `json:"region"`
	UserPoolID    string                             `json:"userPoolId"`
	CallerContext CognitoEventUserPoolsCallerContext `json:"callerContext"`
	UserName      string                             `json:"userName"`
}

// CognitoEventUserPoolsPreSignupRequest contains the request portion of a PreSignup event
type CognitoEventUserPoolsPreSignupRequest struct {
	UserAttributes map[string]string `json:"userAttributes"`
	ValidationData map[string]string `json:"validationData"`
	ClientMetadata map[string]string `json:"clientMetadata"`
}

// CognitoEventUserPoolsPreSignupResponse contains the response portion of a PreSignup event
type CognitoEventUserPoolsPreSignupResponse struct {
	AutoConfirmUser bool `json:"autoConfirmUser"`
	AutoVerifyEmail bool `json:"autoVerifyEmail"`
	AutoVerifyPhone bool `json:"autoVerifyPhone"`
}

// CognitoEventUserPoolsPreAuthenticationRequest contains the request portion of a PreAuthentication event
type CognitoEventUserPoolsPreAuthenticationRequest struct {
	UserAttributes map[string]string `json:"userAttributes"`
	ValidationData map[string]string `json:"validationData"`
}

// CognitoEventUserPoolsPreAuthenticationResponse contains the response portion of a PreAuthentication event
type CognitoEventUserPoolsPreAuthenticationResponse struct {
}

// CognitoEventUserPoolsPostConfirmationRequest contains the request portion of a PostConfirmation event
type CognitoEventUserPoolsPostConfirmationRequest struct {
	UserAttributes map[string]string `json:"userAttributes"`
	ClientMetadata map[string]string `json:"clientMetadata"`
}

// CognitoEventUserPoolsPostConfirmationResponse contains the response portion of a PostConfirmation event
type CognitoEventUserPoolsPostConfirmationResponse struct {
}

// CognitoEventUserPoolsPreTokenGenRequest contains request portion of PreTokenGen event
type CognitoEventUserPoolsPreTokenGenRequest struct {
	UserAttributes     map[string]string  `json:"userAttributes"`
	GroupConfiguration GroupConfiguration `json:"groupConfiguration"`
	ClientMetadata     map[string]string  `json:"clientMetadata"`
}

// CognitoEventUserPoolsPreTokenGenV2Request contains request portion of V2 PreTokenGen event
//
// Deprecated: Use CognitoEventUserPoolsPreTokenGenRequestV2_0 instead
type CognitoEventUserPoolsPreTokenGenV2Request struct {
	UserAttributes     map[string]string  `json:"userAttributes"`
	GroupConfiguration GroupConfiguration `json:"groupConfiguration"`
	ClientMetadata     map[string]string  `json:"clientMetadata,omitempty"`
	Scopes             []string           `json:"scopes"`
}

// CognitoEventUserPoolsPreTokenGenRequestV2_0 contains request portion of V2 PreTokenGen event
type CognitoEventUserPoolsPreTokenGenRequestV2_0 struct {
	UserAttributes     map[string]string      `json:"userAttributes"`
	GroupConfiguration GroupConfigurationV2_0 `json:"groupConfiguration"`
	ClientMetadata     map[string]string      `json:"clientMetadata,omitempty"`
	Scopes             []string               `json:"scopes"`
}

// CognitoEventUserPoolsPreTokenGenResponse contains the response portion of a PreTokenGen event
type CognitoEventUserPoolsPreTokenGenResponse struct {
	ClaimsOverrideDetails ClaimsOverrideDetails `json:"claimsOverrideDetails"`
}

// CognitoEventUserPoolsPreTokenGenV2Response contains the response portion of a V2 PreTokenGen event
//
// Deprecated: Use CognitoEventUserPoolsPreTokenGenResponseV2_0 instead
type CognitoEventUserPoolsPreTokenGenV2Response struct {
	ClaimsAndScopeOverrideDetails ClaimsAndScopeOverrideDetails `json:"claimsAndScopeOverrideDetails"`
}

// CognitoEventUserPoolsPreTokenGenResponseV2_0 contains the response portion of a V2 PreTokenGen event
type CognitoEventUserPoolsPreTokenGenResponseV2_0 struct {
	ClaimsAndScopeOverrideDetails ClaimsAndScopeOverrideDetailsV2_0 `json:"claimsAndScopeOverrideDetails"`
}

// CognitoEventUserPoolsPostAuthenticationRequest contains the request portion of a PostAuthentication event
type CognitoEventUserPoolsPostAuthenticationRequest struct {
	NewDeviceUsed  bool              `json:"newDeviceUsed"`
	UserAttributes map[string]string `json:"userAttributes"`
	ClientMetadata map[string]string `json:"clientMetadata"`
}

// CognitoEventUserPoolsPostAuthenticationResponse contains the response portion of a PostAuthentication event
type CognitoEventUserPoolsPostAuthenticationResponse struct {
}

// CognitoEventUserPoolsMigrateUserRequest contains the request portion of a MigrateUser event
type CognitoEventUserPoolsMigrateUserRequest struct {
	Password       string            `json:"password"`
	ValidationData map[string]string `json:"validationData"`
	ClientMetadata map[string]string `json:"clientMetadata"`
}

// CognitoEventUserPoolsMigrateUserResponse contains the response portion of a MigrateUser event
type CognitoEventUserPoolsMigrateUserResponse struct {
	UserAttributes         map[string]string `json:"userAttributes"`
	FinalUserStatus        string            `json:"finalUserStatus"`
	MessageAction          string            `json:"messageAction"`
	DesiredDeliveryMediums []string          `json:"desiredDeliveryMediums"`
	ForceAliasCreation     bool              `json:"forceAliasCreation"`
}

// ClaimsAndScopeOverrideDetails allows lambda to add, suppress or override V2 claims and scopes in the token
//
// Deprecated: Use ClaimsAndScopeOverrideDetailsV2_0 instead
type ClaimsAndScopeOverrideDetails struct {
	IDTokenGeneration     IDTokenGeneration     `json:"idTokenGeneration"`
	AccessTokenGeneration AccessTokenGeneration `json:"accessTokenGeneration"`
	GroupOverrideDetails  GroupConfiguration    `json:"groupOverrideDetails"`
}

// ClaimsAndScopeOverrideDetailsV2 allows lambda to add, suppress or override V2 claims and scopes in the token
type ClaimsAndScopeOverrideDetailsV2_0 struct {
	IDTokenGeneration     IDTokenGenerationV2_0     `json:"idTokenGeneration"`
	AccessTokenGeneration AccessTokenGenerationV2_0 `json:"accessTokenGeneration"`
	GroupOverrideDetails  GroupConfigurationV2_0    `json:"groupOverrideDetails"`
}

// IDTokenGeneration allows lambda to modify the ID token
type IDTokenGeneration struct {
	ClaimsToAddOrOverride map[string]string `json:"claimsToAddOrOverride"`
	ClaimsToSuppress      []string          `json:"claimsToSuppress"`
}

// IDTokenGenerationV2_0 allows lambda to modify the ID token
type IDTokenGenerationV2_0 struct {
	ClaimsToAddOrOverride map[string]interface{} `json:"claimsToAddOrOverride"`
	ClaimsToSuppress      []string               `json:"claimsToSuppress"`
}

// AccessTokenGeneration allows lambda to modify the access token
//
// Deprecated: Use AccessTokenGenerationV2_0 instead
type AccessTokenGeneration struct {
	ClaimsToAddOrOverride map[string]string `json:"claimsToAddOrOverride"`
	ClaimsToSuppress      []string          `json:"claimsToSuppress"`
	ScopesToAdd           []string          `json:"scopesToAdd"`
	ScopesToSuppress      []string          `json:"scopesToSuppress"`
}

// AccessTokenGenerationV2_0 allows lambda to modify the access token
type AccessTokenGenerationV2_0 struct {
	ClaimsToAddOrOverride map[string]interface{} `json:"claimsToAddOrOverride"`
	ClaimsToSuppress      []string               `json:"claimsToSuppress"`
	ScopesToAdd           []string               `json:"scopesToAdd"`
	ScopesToSuppress      []string               `json:"scopesToSuppress"`
}

// ClaimsOverrideDetails allows lambda to add, suppress or override claims in the token
type ClaimsOverrideDetails struct {
	GroupOverrideDetails  GroupConfiguration `json:"groupOverrideDetails"`
	ClaimsToAddOrOverride map[string]string  `json:"claimsToAddOrOverride"`
	ClaimsToSuppress      []string           `json:"claimsToSuppress"`
}

// GroupConfiguration allows lambda to override groups, roles and set a preferred role
type GroupConfiguration struct {
	GroupsToOverride   []string `json:"groupsToOverride"`
	IAMRolesToOverride []string `json:"iamRolesToOverride"`
	PreferredRole      *string  `json:"preferredRole"`
}

// GroupConfigurationV2_0 allows lambda to override groups, roles and set a preferred role
type GroupConfigurationV2_0 struct {
	GroupsToOverride   []string `json:"groupsToOverride"`
	IAMRolesToOverride []string `json:"iamRolesToOverride"`
	PreferredRole      *string  `json:"preferredRole"`
}

// CognitoEventUserPoolsChallengeResult represents a challenge that is presented to the user in the authentication
// process that is underway, along with the corresponding result.
type CognitoEventUserPoolsChallengeResult struct {
	ChallengeName     string `json:"challengeName"`
	ChallengeResult   bool   `json:"challengeResult"`
	ChallengeMetadata string `json:"challengeMetadata"`
}

// CognitoEventUserPoolsDefineAuthChallengeRequest defines auth challenge request parameters
type CognitoEventUserPoolsDefineAuthChallengeRequest struct {
	UserAttributes map[string]string                       `json:"userAttributes"`
	Session        []*CognitoEventUserPoolsChallengeResult `json:"session"`
	ClientMetadata map[string]string                       `json:"clientMetadata"`
	UserNotFound   bool                                    `json:"userNotFound"`
}

// CognitoEventUserPoolsDefineAuthChallengeResponse defines auth challenge response parameters
type CognitoEventUserPoolsDefineAuthChallengeResponse struct {
	ChallengeName      string `json:"challengeName"`
	IssueTokens        bool   `json:"issueTokens"`
	FailAuthentication bool   `json:"failAuthentication"`
}

// CognitoEventUserPoolsDefineAuthChallenge sent by Amazon Cognito User Pools to initiate custom authentication flow
type CognitoEventUserPoolsDefineAuthChallenge struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsDefineAuthChallengeRequest  `json:"request"`
	Response CognitoEventUserPoolsDefineAuthChallengeResponse `json:"response"`
}

// CognitoEventUserPoolsCreateAuthChallengeRequest defines create auth challenge request parameters
type CognitoEventUserPoolsCreateAuthChallengeRequest struct {
	UserAttributes map[string]string                       `json:"userAttributes"`
	ChallengeName  string                                  `json:"challengeName"`
	Session        []*CognitoEventUserPoolsChallengeResult `json:"session"`
	ClientMetadata map[string]string                       `json:"clientMetadata"`
}

// CognitoEventUserPoolsCreateAuthChallengeResponse defines create auth challenge response rarameters
type CognitoEventUserPoolsCreateAuthChallengeResponse struct {
	PublicChallengeParameters  map[string]string `json:"publicChallengeParameters"`
	PrivateChallengeParameters map[string]string `json:"privateChallengeParameters"`
	ChallengeMetadata          string            `json:"challengeMetadata"`
}

// CognitoEventUserPoolsCreateAuthChallenge sent by Amazon Cognito User Pools to create a challenge to present to the user
type CognitoEventUserPoolsCreateAuthChallenge struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsCreateAuthChallengeRequest  `json:"request"`
	Response CognitoEventUserPoolsCreateAuthChallengeResponse `json:"response"`
}

// CognitoEventUserPoolsVerifyAuthChallengeRequest defines verify auth challenge request parameters
type CognitoEventUserPoolsVerifyAuthChallengeRequest struct {
	UserAttributes             map[string]string `json:"userAttributes"`
	PrivateChallengeParameters map[string]string `json:"privateChallengeParameters"`
	ChallengeAnswer            interface{}       `json:"challengeAnswer"`
	ClientMetadata             map[string]string `json:"clientMetadata"`
}

// CognitoEventUserPoolsVerifyAuthChallengeResponse defines verify auth challenge response parameters
type CognitoEventUserPoolsVerifyAuthChallengeResponse struct {
	AnswerCorrect bool `json:"answerCorrect"`
}

// CognitoEventUserPoolsVerifyAuthChallenge sent by Amazon Cognito User Pools to verify if the response from the end user
// for a custom Auth Challenge is valid or not
type CognitoEventUserPoolsVerifyAuthChallenge struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsVerifyAuthChallengeRequest  `json:"request"`
	Response CognitoEventUserPoolsVerifyAuthChallengeResponse `json:"response"`
}

// CognitoEventUserPoolsCustomMessage is sent by Amazon Cognito User Pools before a verification or MFA message is sent,
// allowing a user to customize the message dynamically.
type CognitoEventUserPoolsCustomMessage struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsCustomMessageRequest  `json:"request"`
	Response CognitoEventUserPoolsCustomMessageResponse `json:"response"`
}

// CognitoEventUserPoolsCustomMessageRequest contains the request portion of a CustomMessage event
type CognitoEventUserPoolsCustomMessageRequest struct {
	UserAttributes    map[string]interface{} `json:"userAttributes"`
	CodeParameter     string                 `json:"codeParameter"`
	UsernameParameter string                 `json:"usernameParameter"`
	ClientMetadata    map[string]string      `json:"clientMetadata"`
}

// CognitoEventUserPoolsCustomMessageResponse contains the response portion of a CustomMessage event
type CognitoEventUserPoolsCustomMessageResponse struct {
	SMSMessage   string `json:"smsMessage"`
	EmailMessage string `json:"emailMessage"`
	EmailSubject string `json:"emailSubject"`
}

// CognitoFederationProviderType is the type of the external identity provider.
type CognitoFederationProviderType string

const (
	CognitoFederationProviderTypeOIDC            CognitoFederationProviderType = "OIDC"
	CognitoFederationProviderTypeSAML            CognitoFederationProviderType = "SAML"
	CognitoFederationProviderTypeFacebook        CognitoFederationProviderType = "Facebook"
	CognitoFederationProviderTypeGoogle          CognitoFederationProviderType = "Google"
	CognitoFederationProviderTypeSignInWithApple CognitoFederationProviderType = "SignInWithApple"
	CognitoFederationProviderTypeLoginWithAmazon CognitoFederationProviderType = "LoginWithAmazon"
)

// CognitoEventUserPoolsInboundFederation is sent by Amazon Cognito User Pools when a user signs in
// through a third-party identity provider, allowing a Lambda to inspect and transform federated user attributes.
type CognitoEventUserPoolsInboundFederation struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsInboundFederationRequest  `json:"request"`
	Response CognitoEventUserPoolsInboundFederationResponse `json:"response"`
}

// CognitoEventUserPoolsInboundFederationRequest contains the request portion of an InboundFederation event
type CognitoEventUserPoolsInboundFederationRequest struct {
	ProviderName string                                           `json:"providerName"`
	ProviderType CognitoFederationProviderType                    `json:"providerType"`
	Attributes   CognitoEventUserPoolsInboundFederationAttributes `json:"attributes"`
}

// CognitoEventUserPoolsInboundFederationAttributes contains the identity provider attributes
type CognitoEventUserPoolsInboundFederationAttributes struct {
	TokenResponse map[string]string `json:"tokenResponse,omitempty"`
	IDToken       map[string]string `json:"idToken,omitempty"`
	UserInfo      map[string]string `json:"userInfo,omitempty"`
	SAMLResponse  map[string]string `json:"samlResponse,omitempty"`
}

// CognitoEventUserPoolsInboundFederationResponse contains the response portion of an InboundFederation event
type CognitoEventUserPoolsInboundFederationResponse struct {
	UserAttributesToMap map[string]string `json:"userAttributesToMap"`
}
