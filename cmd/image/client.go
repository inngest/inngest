package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type client struct {
	base, key, environment string
	http                   *http.Client
}

func newClient(ctx context.Context, cmd *cli.Command) (*client, error) {
	base := cmd.String("api-url")
	if base == "" {
		issuer, err := cliauth.Issuer()
		if err != nil {
			return nil, err
		}
		base = cliauth.Resource(issuer)
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid image API URL")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback())) {
		return nil, errors.New("image API requires HTTPS or loopback HTTP")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/v2") {
		u.Path += "/v2"
	}
	c := &client{base: u.String(), key: cmd.String("api-key"), environment: cmd.String("env"), http: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	if c.key == "" {
		manager, err := cliauth.NewManager()
		if err != nil {
			return nil, err
		}
		c.key, _, err = manager.AccessToken(ctx, c.base)
		if err != nil {
			return nil, errors.New("run inngest login or set INNGEST_API_KEY to build images")
		}
	}
	return c, nil
}

type apiError struct{ status int }

func (e apiError) Error() string { return fmt.Sprintf("image API returned HTTP %d", e.status) }

func (c *client) request(ctx context.Context, method, path string, body, output proto.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = protojson.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	if c.environment != "" {
		req.Header.Set("X-Inngest-Env", c.environment)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return errors.New("image API response was not confirmed; retry with the same upload ID")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return apiError{response.StatusCode}
	}
	if output == nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	return (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, output)
}

func (c *client) upload(ctx context.Context, rawURL string, headers map[string]string, file *os.File) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	return c.uploadRange(ctx, rawURL, headers, file, 0, stat.Size(), true)
}

func (c *client) uploadGrant(ctx context.Context, grant *apiv2.ImageUploadGrant, file *os.File) error {
	if grant.GetAlreadyUploaded() {
		return nil
	}
	if len(grant.GetParts()) == 0 {
		return c.upload(ctx, grant.Url, grant.Headers, file)
	}
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	var offset int64
	for i, part := range grant.Parts {
		if part.Number != int32(i+1) || part.Offset != offset || part.SizeBytes <= 0 || part.SizeBytes > stat.Size()-offset {
			return errors.New("invalid image upload byte ranges")
		}
		offset += part.SizeBytes
	}
	if offset != stat.Size() {
		return errors.New("image upload ranges do not cover the archive")
	}
	for _, part := range grant.Parts {
		if err := c.uploadRange(ctx, part.Url, part.Headers, file, part.Offset, part.SizeBytes, false); err != nil {
			return err
		}
	}
	return nil
}

func (c *client) uploadRange(ctx context.Context, rawURL string, headers map[string]string, file *os.File, offset, size int64, single bool) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return errors.New("image upload requires an HTTPS storage grant")
	}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), io.NewSectionReader(file, offset, size))
		if err != nil {
			return err
		}
		req.ContentLength = size
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		// Storage requests carry only signed-grant headers, never API credentials.
		response, err := c.http.Do(req)
		status := 0
		if response != nil {
			status = response.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
			_ = response.Body.Close()
		}
		if err == nil && ((single && status == http.StatusPreconditionFailed) || (status >= 200 && status < 300)) {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt >= 2 || (err == nil && status < 500) {
			if err != nil {
				return errors.New("image upload response was not confirmed; retry with the same upload ID")
			}
			return fmt.Errorf("image upload returned HTTP %d", status)
		}
		timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
