package httpdriver

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	headerSDK = "x-inngest-sdk"
)

// getSDKVersion parses the SDK version from the response header.
func getSDKVersion(resp *http.Response) (string, error) {
	raw := resp.Header.Get(headerSDK)
	versionHeader := strings.Split(raw, ":")
	if len(versionHeader) != 2 {
		return "", fmt.Errorf("unexpected SDK header: %s", raw)
	}

	return versionHeader[1], nil
}
