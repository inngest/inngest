package imagecmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/inngest/version"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	apiTimeout       = 30 * time.Second
	maxResponseBytes = 1 << 20
)

// client calls the v2 builds API with the login from `inngest login`.
type client struct {
	baseURL string
	// token returns an access token for each request.  a deploy can run
	// longer than one access token lasts, and the saved login refreshes
	// the token only when it is asked for one.
	token func(context.Context) (string, error)
	// env selects the environment when the login covers more than one.  it
	// comes from INNGEST_ENV, the same variable `inngest api` reads.
	env  string
	http *http.Client
}

func (c *client) createBuild(ctx context.Context, req *apiv2.CreateBuildRequest) (*apiv2.CreateBuildResponse, error) {
	resp := &apiv2.CreateBuildResponse{}
	err := c.do(ctx, http.MethodPost, "/builds/"+url.PathEscape(req.GetName()), req, resp)
	return resp, err
}

func (c *client) commitBuild(ctx context.Context, name, buildID string) (*apiv2.Build, error) {
	req := &apiv2.CommitBuildRequest{
		BuildId: buildID,
		Name:    name,
	}
	resp := &apiv2.BuildResponse{}
	path := "/builds/" + url.PathEscape(name) + "/" + url.PathEscape(buildID) + "/commit"
	if err := c.do(ctx, http.MethodPost, path, req, resp); err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}

func (c *client) getBuild(ctx context.Context, buildID string) (*apiv2.Build, error) {
	resp := &apiv2.BuildResponse{}
	if err := c.do(ctx, http.MethodGet, "/builds/"+url.PathEscape(buildID), nil, resp); err != nil {
		return nil, err
	}
	return resp.GetData(), nil
}

func (c *client) do(ctx context.Context, method, path string, in, out proto.Message) error {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	var body io.Reader
	if in != nil {
		encoded, err := protojson.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "inngest-cli/"+version.Version)
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if c.env != "" {
		req.Header.Set("X-Inngest-Env", c.env)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(respBody)))
	}
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(respBody, out)
}

// uploadArchive sends the archive to the signed URL from createBuild.
// storage rejects the PUT when a header in headers is missing or changed,
// or when the body length or SHA256 differs from the values in createBuild.
func uploadArchive(ctx context.Context, uploadURL string, headers map[string]string, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, f)
	if err != nil {
		return err
	}
	req.ContentLength = info.Size()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// the upload has no timeout because an archive can be 5 GiB.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

// fileDigest returns the size and lowercase hex SHA256 of the file.
func fileDigest(path string) (int64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return 0, "", err
	}
	return size, hex.EncodeToString(h.Sum(nil)), nil
}
