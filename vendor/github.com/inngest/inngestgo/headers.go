package inngestgo

import (
	"fmt"
	"net/http"
)

const (
	HeaderKeyAuthorization      = "Authorization"
	HeaderKeyContentType        = "Content-Type"
	HeaderKeyEnv                = "X-Inngest-Env"
	HeaderKeyExpectedServerKind = "X-Inngest-Expected-Server-Kind"
	HeaderKeyNoRetry            = "X-Inngest-No-Retry"
	HeaderKeyReqVersion         = "x-inngest-req-version"
	HeaderKeyRetryAfter         = "Retry-After"
	HeaderKeySDK                = "X-Inngest-SDK"
	HeaderKeyServerKind         = "X-Inngest-Server-Kind"
	HeaderKeySignature          = "X-Inngest-Signature"
	HeaderKeySyncKind           = "x-inngest-sync-kind"
	HeaderKeyUserAgent          = "User-Agent"
)

var (
	HeaderValueSDK = fmt.Sprintf("%s:v%s", SDKLanguage, SDKVersion)
)

func SetBasicRequestHeaders(req *http.Request) {
	req.Header.Set(HeaderKeyContentType, "application/json")
	req.Header.Set(HeaderKeySDK, HeaderValueSDK)
	req.Header.Set(HeaderKeyUserAgent, HeaderValueSDK)
}

func SetBasicResponseHeaders(w http.ResponseWriter) {
	w.Header().Set(HeaderKeyContentType, "application/json")
	w.Header().Set(HeaderKeyReqVersion, executionVersionV2)
	w.Header().Set(HeaderKeySDK, HeaderValueSDK)
	w.Header().Set(HeaderKeyUserAgent, HeaderValueSDK)
}
