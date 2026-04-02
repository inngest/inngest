package events

// SecretsManagerSecretRotationEvent is the event passed to a Lambda function to handle
// automatic secret rotation.
//
// https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotating-secrets.html#rotate-secrets_how
type SecretsManagerSecretRotationEvent struct {
	Step               string `json:"Step"`
	SecretID           string `json:"SecretId"`
	ClientRequestToken string `json:"ClientRequestToken"`
	RotationToken      string `json:"RotationToken"`
}
