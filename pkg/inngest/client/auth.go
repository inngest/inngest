package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

func (c httpClient) Login(ctx context.Context, email, password string) ([]byte, error) {
	input := map[string]string{
		"email":    email,
		"password": password,
	}
	buf := jsonBuffer(ctx, input)

	req, err := c.NewRequest(http.MethodPost, "/v1/login", buf)
	if err != nil {
		return nil, fmt.Errorf("error creating login request: %s", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing login request: %s", err)
	}
	defer resp.Body.Close()
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %s", err)
	}

	type response struct {
		Message string
		JWT     string
	}

	r := &response{}
	if err = json.Unmarshal(byt, r); err != nil {
		return nil, fmt.Errorf("invalid json response: %w: \n%s", err, string(byt))
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", r.Message)
	}

	return []byte(r.JWT), nil
}

func (c httpClient) StartDeviceLogin(ctx context.Context, clientID uuid.UUID) (*StartDeviceLoginResponse, error) {
	if clientID == uuid.Nil {
		return nil, fmt.Errorf("Please provide a valid client ID")
	}

	req, err := c.NewRequest(http.MethodPost, fmt.Sprintf("/v2/login/device/new?client_id=%s", clientID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating login request: %s", err)
	}
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing login request: %s", err)
	}
	defer resp.Body.Close()
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Unable to start the device login flow: %s", byt)
	}
	r := &StartDeviceLoginResponse{}
	if err = json.Unmarshal(byt, r); err != nil {
		return nil, fmt.Errorf("invalid json response: %w: \n%s", err, string(byt))
	}
	return r, nil
}

func (c httpClient) PollDeviceLogin(ctx context.Context, clientID, deviceCode uuid.UUID) (*DeviceLoginResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID.String())
	data.Set("device_code", deviceCode.String())
	req, err := c.NewRequest(http.MethodPost, "/v2/login/device/poll", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating login request: %s", err)
	}
	// Cancellation (Ctrl-C, code expiry) must abort the server-side long-poll.
	req = req.WithContext(ctx)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing login request: %s", err)
	}
	defer resp.Body.Close()
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %s", err)
	}
	r := &DeviceLoginResponse{}
	if err = json.Unmarshal(byt, r); err != nil {
		return nil, fmt.Errorf("invalid json response: %w: \n%s", err, string(byt))
	}
	return r, nil
}

// RevokeDeviceLogin revokes the API key the client is configured with, so a
// logged-out credential can no longer be used.
func (c httpClient) RevokeDeviceLogin(ctx context.Context) error {
	req, err := c.NewRequest(http.MethodPost, "/v2/login/device/revoke", nil)
	if err != nil {
		return fmt.Errorf("error creating revoke request: %s", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.creds))
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("error performing revoke request: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		byt, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unable to revoke the API key: %s", byt)
	}
	return nil
}

type StartDeviceLoginResponse struct {
	DeviceCode      uuid.UUID `json:"device_code"`
	ExpiresIn       int       `json:"expires_in"`
	Interval        int       `json:"interval"`
	UserCode        string    `json:"user_code"`
	VerificationURL string    `json:"verification_url"`
}

type DeviceLoginResponse struct {
	Error       string    `json:"error"`
	AccessToken string    `json:"access_token"`
	AccountID   uuid.UUID `json:"account_id"`
	AccountName string    `json:"account_name"`
	UserID      uuid.UUID `json:"user_id"`
	Env         string    `json:"env"`
}
