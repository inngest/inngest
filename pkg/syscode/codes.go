package syscode

const (
	CodeBatchSizeInvalid          = "batch_size_invalid"
	CodeBatchTimeoutInvalid       = "batch_timeout_invalid"
	CodeComboUnsupported          = "combo_unsupported"
	CodeConcurrencyLimitInvalid   = "concurrency_limit_invalid"
	CodeConfigInvalid             = "config_invalid"
	CodeHTTPMissingHeader         = "http_missing_header"
	CodeHTTPNotOK                 = "http_not_ok"
	CodeHTTPUnreachable           = "http_unreachable"
	CodeNotSDK                    = "not_sdk"
	CodeOutputTooLarge            = "output_too_large"
	CodeSigVerificationFailed     = "sig_verification_failed"
	CodeUnknown                   = "unknown"
	CodeBatchKeyExpressionInvalid = "batch_key_expression_invalid"
	CodeSyncAlreadyPending        = "sync_already_pending"
	CodePlanUpgradeRequired       = "plan_upgrade_required"

	// Connect
	CodeConnectWorkerHelloTimeout             = "connect_worker_hello_timeout"
	CodeConnectWorkerHelloInvalidMsg          = "connect_worker_hello_invalid_msg"
	CodeConnectWorkerHelloInvalidPayload      = "connect_worker_hello_invalid_payload"
	CodeConnectAuthFailed                     = "connect_authentication_failed"
	CodeConnectConnNotSaved                   = "connect_connection_not_saved"
	CodeConnectInternal                       = "connect_internal_error"
	CodeConnectGatewayClosing                 = "connect_gateway_closing"
	CodeConnectRunInvalidMessage              = "connect_run_invalid_message"
	CodeConnectInvalidFunctionConfig          = "connect_invalid_function_config"
	CodeConnectWorkerRequestAckInvalidPayload = "connect_worker_request_ack_invalid_payload"
)
