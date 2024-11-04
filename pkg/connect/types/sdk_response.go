package types

import "net/http"

type SdkResponseStatus int

const (
	SdkResponseStatusNotCompleted SdkResponseStatus = http.StatusPartialContent
	SdkResponseStatusDone         SdkResponseStatus = http.StatusOK
	SdkResponseStatusError        SdkResponseStatus = http.StatusInternalServerError
)

type SdkResponse struct {
	RequestId string `json:"replyId"`

	Status SdkResponseStatus `json:"status"`
	Body   []byte            `json:"body"`

	// These are modeled after the headers for code reuse in httpdriver.ShouldRetry
	NoRetry    string `json:"no_retry"`
	RetryAfter string `json:"retry_after"`
	SdkVersion string `json:"sdk_version"`
}
