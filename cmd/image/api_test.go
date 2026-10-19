package imagecmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestBuildAPI(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "image.tar")
	require.NoError(t, os.WriteFile(archive, []byte("image bytes"), 0o600))
	size, digest, err := fileDigest(archive)
	require.NoError(t, err)
	raw, _ := hex.DecodeString(digest)
	checksum := base64.StdEncoding.EncodeToString(raw)

	var uploaded []byte
	polls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/builds/app", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		require.Equal(t, "staging", r.Header.Get("X-Inngest-Env"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, digest, body["uploadSha256"])
		require.Equal(t, "amd64", body["architecture"])
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"b1","status":"pending"},"upload_url":"` + "http://" + r.Host + `/upload","upload_headers":{"x-amz-checksum-sha256":"` + checksum + `"}}`))
	})
	mux.HandleFunc("PUT /upload", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, checksum, r.Header.Get("x-amz-checksum-sha256"))
		require.Equal(t, size, r.ContentLength)
		uploaded, _ = io.ReadAll(r.Body)
	})
	mux.HandleFunc("POST /v2/builds/app/b1/commit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"id":"b1","status":"uploaded"}}`))
	})
	mux.HandleFunc("GET /v2/builds/b1", func(w http.ResponseWriter, r *http.Request) {
		polls++
		_, _ = w.Write([]byte(`{"data":{"id":"b1","status":"ready","artifact_digest":"abc"}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx := context.Background()
	api := &client{
		baseURL: srv.URL + "/v2",
		token:   "token",
		env:     "staging",
		http:    srv.Client(),
	}

	created, err := api.createBuild(ctx, &apiv2.CreateBuildRequest{
		Name:            "app",
		Tag:             "v1",
		Architecture:    "amd64",
		UploadSizeBytes: size,
		UploadSha256:    digest,
	})
	require.NoError(t, err)
	require.Equal(t, "b1", created.GetData().GetId())

	require.NoError(t, uploadArchive(ctx, created.GetUploadUrl(), created.GetUploadHeaders(), archive))
	sum := sha256.Sum256(uploaded)
	require.Equal(t, digest, hex.EncodeToString(sum[:]))

	b, err := api.commitBuild(ctx, "app", "b1")
	require.NoError(t, err)
	require.Equal(t, "uploaded", b.GetStatus())

	b, err = waitForBuild(ctx, api, b)
	require.NoError(t, err)
	require.Equal(t, "abc", b.GetArtifactDigest())
	require.Equal(t, 1, polls)
}

func TestWaitForBuildFailed(t *testing.T) {
	_, err := waitForBuild(context.Background(), nil, &apiv2.Build{
		Id:     "b1",
		Status: buildFailed,
		Error:  "bad layer",
	})
	require.ErrorContains(t, err, "bad layer")
}
