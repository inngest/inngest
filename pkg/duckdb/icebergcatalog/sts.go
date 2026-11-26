package icebergcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/sync/singleflight"
)

// AssumeRoleAPI is the one sts.Client method STSVendor uses.
type AssumeRoleAPI interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// STSVendor vends S3 creds via STS AssumeRole w/ a session policy that only
// allows the table's own prefix, so a client can't read or write any other
// table w/ them. It works against AWS or any S3-compatible STS that
// enforces session policies (IE SeaweedFS).
type STSVendor struct {
	STS     AssumeRoleAPI
	RoleARN string
	// Duration is the creds' lifetime (AWS' minimum is 15m, its default 1h).
	Duration time.Duration
	// Region, Endpoint and PathStyle are passed through to the client as
	// `s3.region`, `s3.endpoint` and `s3.path-style-access`. Endpoint is
	// only needed for a non-AWS S3 (IE SeaweedFS).
	Region    string
	Endpoint  string
	PathStyle bool
	// ReadOnly vends read-only creds (IE for a reader that never commits).
	ReadOnly bool

	// Now is the clock (time.Now if nil), for tests.
	Now func() time.Time

	mu    sync.Mutex
	cache map[string]cachedCreds
	sf    singleflight.Group
}

// cachedCreds are one location's creds. duckdb loads a table on every
// query, so w/o reuse every query would be an STS call (which AWS
// throttles per account).
type cachedCreds struct {
	props   iceberg.Properties
	refresh time.Time
}

var _ CredentialVendor = (*STSVendor)(nil)
var _ readOnlyVendor = (*STSVendor)(nil)

func (v *STSVendor) readOnly() CredentialVendor {
	if v.ReadOnly {
		return v
	}
	return &STSVendor{
		STS:       v.STS,
		RoleARN:   v.RoleARN,
		Duration:  v.Duration,
		Region:    v.Region,
		Endpoint:  v.Endpoint,
		PathStyle: v.PathStyle,
		ReadOnly:  true,
		Now:       v.Now,
	}
}

func (v *STSVendor) VendCredentials(ctx context.Context, ident table.Identifier, location string) (iceberg.Properties, error) {
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	v.mu.Lock()
	c, ok := v.cache[location]
	v.mu.Unlock()
	if ok && now().Before(c.refresh) {
		return maps.Clone(c.props), nil
	}

	ch := v.sf.DoChan(location, func() (any, error) {
		v.mu.Lock()
		c, ok := v.cache[location]
		v.mu.Unlock()
		if ok && now().Before(c.refresh) {
			return c.props, nil
		}
		sfCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		props, exp, err := v.assumeRole(sfCtx, ident, location)
		if err != nil {
			return nil, err
		}
		// Reused for the first half of their life, so a client always gets at
		// least half a lifetime before it has to refresh.
		issued := now()
		refresh := issued
		if !exp.IsZero() {
			refresh = issued.Add(exp.Sub(issued) / 2)
		}
		v.mu.Lock()
		if v.cache == nil {
			v.cache = map[string]cachedCreds{}
		}
		// Evict expired entries so cache memory stays bounded.
		for k, entry := range v.cache {
			if !issued.Before(entry.refresh) {
				delete(v.cache, k)
			}
		}
		if issued.Before(refresh) {
			v.cache[location] = cachedCreds{props: props, refresh: refresh}
		}
		v.mu.Unlock()
		return props, nil
	})
	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return maps.Clone(res.Val.(iceberg.Properties)), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (v *STSVendor) assumeRole(ctx context.Context, ident table.Identifier, location string) (iceberg.Properties, time.Time, error) {
	if v.Duration < 0 {
		return nil, time.Time{}, fmt.Errorf("icebergcatalog: invalid STS duration %v", v.Duration)
	}
	if v.Duration > 0 {
		if v.Duration < time.Second {
			return nil, time.Time{}, fmt.Errorf("icebergcatalog: STS duration %v must be at least 1s", v.Duration)
		}
		if v.Duration/time.Second > math.MaxInt32 {
			return nil, time.Time{}, fmt.Errorf("icebergcatalog: STS duration %v is too large", v.Duration)
		}
	}
	policy, err := tablePolicy(location, v.ReadOnly)
	if err != nil {
		return nil, time.Time{}, err
	}
	in := &sts.AssumeRoleInput{
		RoleArn:         aws.String(v.RoleARN),
		RoleSessionName: aws.String(sessionName(ident)),
		Policy:          aws.String(policy),
	}
	if v.Duration > 0 {
		in.DurationSeconds = aws.Int32(int32(v.Duration / time.Second))
	}
	out, err := v.STS.AssumeRole(ctx, in)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("icebergcatalog: assuming role for %s: %w", strings.Join(ident, "."), err)
	}
	c := out.Credentials
	if c == nil || c.AccessKeyId == nil || c.SecretAccessKey == nil || c.SessionToken == nil {
		return nil, time.Time{}, errors.New("icebergcatalog: AssumeRole returned no credentials")
	}
	props := iceberg.Properties{
		"s3.access-key-id":     *c.AccessKeyId,
		"s3.secret-access-key": *c.SecretAccessKey,
		"s3.session-token":     *c.SessionToken,
	}
	var exp time.Time
	if c.Expiration != nil {
		exp = *c.Expiration
		props["s3.session-token-expires-at-ms"] = strconv.FormatInt(exp.UnixMilli(), 10)
	}
	setS3EndpointProps(props, v.Region, v.Endpoint, v.PathStyle)
	return props, exp, nil
}

// sessionName is the table's name in STS' allowed charset ([\w+=,.@-],
// 2-64 chars), so the session shows up recognizably in audit logs.
func sessionName(ident table.Identifier) string {
	name := []byte("iceberg-" + strings.Join(ident, "."))
	for i, c := range name {
		ok := c == '_' || c == '+' || c == '=' || c == ',' || c == '.' || c == '@' || c == '-' ||
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !ok {
			name[i] = '_'
		}
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return string(name)
}

// tablePolicy is an IAM session policy for an `s3://bucket/prefix` table
// location: objects under the prefix, and listing only that prefix.
func tablePolicy(location string, readOnly bool) (string, error) {
	if strings.ContainsAny(location, iamUnsafeChars) {
		return "", fmt.Errorf("icebergcatalog: invalid characters in table location %q", location)
	}
	u, err := url.Parse(location)
	if err != nil || (u.Scheme != "s3" && u.Scheme != "s3a") || u.Host == "" || u.User != nil {
		return "", fmt.Errorf("icebergcatalog: can't vend S3 creds for table location %q", location)
	}
	if strings.ContainsAny(u.Host, iamUnsafeChars) || strings.ContainsAny(u.Path, iamUnsafeChars) {
		return "", fmt.Errorf("icebergcatalog: invalid characters in table location %q", location)
	}
	prefix := strings.Trim(u.Path, "/")
	if prefix == "" {
		return "", fmt.Errorf("icebergcatalog: table location %q is a whole bucket", location)
	}
	actions := []string{"s3:GetObject"}
	if !readOnly {
		actions = append(actions, "s3:PutObject", "s3:DeleteObject", "s3:AbortMultipartUpload")
	}
	type stmt struct {
		Effect    string         `json:"Effect"`
		Action    []string       `json:"Action"`
		Resource  []string       `json:"Resource"`
		Condition map[string]any `json:"Condition,omitempty"`
	}
	doc := struct {
		Version   string `json:"Version"`
		Statement []stmt `json:"Statement"`
	}{
		Version: "2012-10-17",
		Statement: []stmt{
			{Effect: "Allow", Action: actions, Resource: []string{"arn:aws:s3:::" + u.Host + "/" + prefix + "/*"}},
			{
				Effect: "Allow", Action: []string{"s3:ListBucket"}, Resource: []string{"arn:aws:s3:::" + u.Host},
				Condition: map[string]any{"StringLike": map[string]any{"s3:prefix": []string{prefix, prefix + "/*"}}},
			},
		},
	}
	b, err := json.Marshal(doc)
	return string(b), err
}
