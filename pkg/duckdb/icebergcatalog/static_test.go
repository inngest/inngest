package icebergcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticVendor(t *testing.T) {
	v := &StaticVendor{AccessKeyID: "AK", SecretAccessKey: "SK", Region: "auto", Endpoint: "https://fly.storage.tigris.dev"}
	props, err := v.VendCredentials(t.Context(), []string{"inngest", "runs"}, "s3://bucket/wh/inngest.db/runs")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"s3.access-key-id": "AK", "s3.secret-access-key": "SK",
		"s3.region": "auto", "s3.endpoint": "https://fly.storage.tigris.dev",
	}, map[string]string(props))

	v.SessionToken, v.PathStyle = "TOK", true
	props, err = v.VendCredentials(t.Context(), []string{"t"}, "s3://bucket/t")
	require.NoError(t, err)
	require.Equal(t, "TOK", props["s3.session-token"])
	require.Equal(t, "true", props["s3.path-style-access"])
}
