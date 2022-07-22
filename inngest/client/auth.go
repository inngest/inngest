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

	req, err := c.NewRequest(http.MethodPost, fmt.Sprintf("/v1/login/device/new?client_id=%s", clientID), nil)
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
	req, err := c.NewRequest(http.MethodPost, "/v1/login/device/poll", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating login request: %s", err)
	}
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

type StartDeviceLoginResponse struct {
	DeviceCode      uuid.UUID `json:"device_code"`
	ExpiresIn       int       `json:"expires_in"`
	Interval        int       `json:"interval"`
	UserCode        string    `json:"user_code"`
	VerificationURL string    `json:"verification_url"`
}

type DeviceLoginResponse struct {
	Error       string `json:"error"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expires     int    `json:"expires"`
}
