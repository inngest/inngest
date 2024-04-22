package inngestgo

import (
	"fmt"
	"net/http"
)

func fetchWithAuthFallback(
	createRequest func() (*http.Request, error),
	signingKey string,
	signingKeyFallback string,
) (*http.Response, error) {
	req, err := createRequest()
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if signingKey != "" {
		key, err := hashedSigningKey([]byte(signingKey))
		if err != nil {
			return nil, fmt.Errorf("error creating signing key: %w", err)
		}
		req.Header.Set(HeaderKeyAuthorization, fmt.Sprintf("Bearer %s", string(key)))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && signingKeyFallback != "" {
		// Try again with the signing key fallback
		req, err := createRequest()
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		_ = resp.Body.Close()

		key, err := hashedSigningKey([]byte(signingKeyFallback))
		if err != nil {
			return nil, fmt.Errorf("error creating signing key: %w", err)
		}
		req.Header.Set(HeaderKeyAuthorization, fmt.Sprintf("Bearer %s", string(key)))

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %w", err)
		}
	}

	return resp, nil
}
