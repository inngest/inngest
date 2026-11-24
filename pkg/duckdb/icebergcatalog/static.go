package icebergcatalog

import (
	"context"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
)

// StaticVendor vends the same S3 creds for every table, IE a bucket key for
// an object store w/o STS (Tigris, R2). Nothing is scoped to the table, so
// isolation is only the sandbox's `allowed_directories`, but the key lives
// in the catalog's config instead of every duckdb's.
type StaticVendor struct {
	AccessKeyID     string
	SecretAccessKey string
	// SessionToken is optional, for already temporary creds.
	SessionToken string
	// Region, Endpoint and PathStyle are passed through as in STSVendor.
	Region    string
	Endpoint  string
	PathStyle bool
}

var _ CredentialVendor = (*StaticVendor)(nil)

func (v *StaticVendor) VendCredentials(context.Context, table.Identifier, string) (iceberg.Properties, error) {
	props := iceberg.Properties{
		"s3.access-key-id":     v.AccessKeyID,
		"s3.secret-access-key": v.SecretAccessKey,
	}
	if v.SessionToken != "" {
		props["s3.session-token"] = v.SessionToken
	}
	setS3EndpointProps(props, v.Region, v.Endpoint, v.PathStyle)
	return props, nil
}

// setS3EndpointProps sets the non-secret FileIO properties a client needs to
// reach the store.
func setS3EndpointProps(props iceberg.Properties, region, endpoint string, pathStyle bool) {
	if region != "" {
		props["s3.region"] = region
	}
	if endpoint != "" {
		props["s3.endpoint"] = endpoint
	}
	if pathStyle {
		props["s3.path-style-access"] = "true"
	}
}
