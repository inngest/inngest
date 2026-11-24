// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package s3

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	internalaws "github.com/apache/iceberg-go/internal/awsconfig"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/auth/bearer"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// ParseAWSConfig parses S3 properties and returns a configuration.
func ParseAWSConfig(ctx context.Context, props map[string]string) (*aws.Config, error) {
	// Remote S3 request signing is not implemented yet.
	if v, ok := props[io.S3RemoteSigningEnabled]; ok {
		if enabled, err := strconv.ParseBool(v); err == nil && enabled {
			return nil, errors.New("remote S3 request signing is not supported")
		}
	}

	opts := []func(*config.LoadOptions) error{}
	var httpClient *awshttp.BuildableClient

	if tok, ok := props["token"]; ok {
		opts = append(opts, config.WithBearerAuthTokenProvider(
			&bearer.StaticTokenProvider{Token: bearer.Token{Value: tok}},
		))
	}

	if region, ok := props[io.S3Region]; ok {
		opts = append(opts, config.WithRegion(region))
	} else if region, ok := props[io.S3ClientRegion]; ok {
		opts = append(opts, config.WithRegion(region))
	}

	accessKey, secretAccessKey := props[io.S3AccessKeyID], props[io.S3SecretAccessKey]
	token := props[io.S3SessionToken]
	if err := internalaws.ValidateStaticCredentials(io.S3AccessKeyID, io.S3SecretAccessKey, io.S3SessionToken, accessKey, secretAccessKey, token); err != nil {
		return nil, err
	}
	if accessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey, secretAccessKey, token,
		)))
	}

	if proxy, ok := props[io.S3ProxyURI]; ok {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid s3 proxy url %q: %w", proxy, err)
		}

		httpClient = newS3BuildableClient().WithTransportOptions(
			func(t *http.Transport) {
				t.Proxy = http.ProxyURL(proxyURL)
			},
		)
	}

	if timeout, ok := props[io.S3ConnectTimeout]; ok {
		duration, err := parseS3ConnectTimeout(timeout)
		if err != nil {
			return nil, err
		}

		if httpClient == nil {
			httpClient = newS3BuildableClient()
		}
		httpClient = httpClient.WithDialerOptions(func(d *net.Dialer) {
			d.Timeout = duration
		})
	}

	if httpClient != nil {
		opts = append(opts, config.WithHTTPClient(httpClient))
	}

	awscfg := new(aws.Config)
	var err error
	*awscfg, err = config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return awscfg, nil
}

func parseS3ConnectTimeout(timeout string) (time.Duration, error) {
	var duration time.Duration
	if seconds, err := strconv.ParseFloat(timeout, 64); err == nil {
		duration = time.Duration(seconds * float64(time.Second))
	} else {
		parsedDuration, err := time.ParseDuration(timeout)
		if err != nil {
			return 0, fmt.Errorf("invalid s3.connect-timeout %q: must be seconds as a number or a Go duration string", timeout)
		}
		duration = parsedDuration
	}

	if duration <= 0 {
		return 0, errors.New("s3.connect-timeout must be a positive duration")
	}

	return duration, nil
}

// S3 transport tuning shared by all S3 BuildableClient paths.
const (
	s3MaxIdleConns        = 256
	s3MaxIdleConnsPerHost = 256
	s3MaxConnsPerHost     = 2048
	s3IdleConnTimeout     = 90 * time.Second
)

// newS3BuildableClient returns an AWS buildable HTTP client with the S3
// transport tuning applied. Subsequent WithTransportOptions/WithDialerOptions
// calls preserve this tuning, since the builder clones the transport forward.
func newS3BuildableClient() *awshttp.BuildableClient {
	return awshttp.NewBuildableClient().WithTransportOptions(applyS3TransportTuning)
}

func applyS3TransportTuning(t *http.Transport) {
	t.MaxIdleConns = s3MaxIdleConns
	t.MaxIdleConnsPerHost = s3MaxIdleConnsPerHost
	t.MaxConnsPerHost = s3MaxConnsPerHost
	t.IdleConnTimeout = s3IdleConnTimeout
}

// resolveUsePathStyle determines whether the S3 client should use
// path-style addressing. It defaults to virtual-hosted style for
// standard AWS S3 and path-style for custom endpoints (e.g. MinIO).
// The s3.force-virtual-addressing property can override either default.
func resolveUsePathStyle(endpoint string, props map[string]string) bool {
	usePathStyle := endpoint != ""
	if forceVirtual, ok := props[io.S3ForceVirtualAddressing]; ok {
		if cfgForceVirtual, err := strconv.ParseBool(forceVirtual); err == nil {
			usePathStyle = !cfgForceVirtual
		}
	}

	return usePathStyle
}

// resolveS3AWSConfig returns the AWS config for the S3 FileIO, preferring an
// ambient context config but letting explicit s3.* credentials override it.
func resolveS3AWSConfig(ctx context.Context, props map[string]string) (*aws.Config, error) {
	if err := internalaws.ValidateStaticCredentials(
		io.S3AccessKeyID, io.S3SecretAccessKey, io.S3SessionToken,
		props[io.S3AccessKeyID], props[io.S3SecretAccessKey], props[io.S3SessionToken],
	); err != nil {
		return nil, err
	}

	var (
		base *aws.Config
		err  error
	)
	if v := utils.GetAwsConfig(ctx); v != nil {
		base = v
	} else if base, err = ParseAWSConfig(ctx, props); err != nil {
		return nil, err
	}

	// Always copy so overrides (and the caller's later HTTPClient assignment)
	// never touch a shared context config.
	cfg := *base

	// Explicit region overrides the context config; an unset region falls through.
	if r := props[io.S3Region]; r != "" {
		cfg.Region = r
	} else if r := props[io.S3ClientRegion]; r != "" {
		cfg.Region = r
	}

	// A complete explicit key pair overrides the credentials.
	if props[io.S3AccessKeyID] != "" && props[io.S3SecretAccessKey] != "" {
		cfg.Credentials = credentials.NewStaticCredentialsProvider(
			props[io.S3AccessKeyID], props[io.S3SecretAccessKey], props[io.S3SessionToken],
		)
	}

	return &cfg, nil
}

func createS3Bucket(ctx context.Context, parsed *url.URL, props map[string]string) (*blob.Bucket, error) {
	awscfg, err := resolveS3AWSConfig(ctx, props)
	if err != nil {
		return nil, err
	}

	// Default HTTP client when not configured: use the SDK buildable client so
	// proxy, TLS, dial, and HTTP/2 behavior match the usual AWS defaults, with
	// the S3 transport tuning applied (see applyS3TransportTuning).
	if awscfg.HTTPClient == nil {
		awscfg.HTTPClient = newS3BuildableClient()
	}

	endpoint, ok := props[io.S3EndpointURL]
	if !ok {
		endpoint = os.Getenv("AWS_S3_ENDPOINT")
	}

	compatMode := s3CompatModeEnabled(props)

	client := s3.NewFromConfig(*awscfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		if compatMode {
			// Compat mode trades checksums for interoperability: endpoints such as
			// GCS's S3 interop layer reject the x-amz-checksum-* family outright, so
			// no request checksum is sent on object writes and any caller-supplied
			// digest is silently dropped (see clearS3WriteChecksumFields).
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			o.APIOptions = append(o.APIOptions, stripS3InputChecksumAlgorithm)
			o.APIOptions = append(o.APIOptions, stripGCSIncompatibleSignedHeaders)
		}
		o.UsePathStyle = resolveUsePathStyle(endpoint, props)
		o.DisableLogOutputChecksumValidationSkipped = true
	})

	// Create a *blob.Bucket.
	bucket, err := s3blob.OpenBucketV2(ctx, client, parsed.Host, nil)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// awsChecksumSetupInputContextID is the ID of the SDK middleware that records
// which input checksum algorithm the SDK should compute for itself. That context
// value drives the SDK-computed checksum: the x-amz-checksum-* header it derives
// from the payload, the x-amz-trailer and aws-chunked framing, and the
// STREAMING- payload hash.
//
// It does not cover checksums the caller supplies up front. The modeled
// Checksum* input fields are bound directly to x-amz-checksum-* headers by the
// serializers without consulting this context, so they have to be cleared on the
// input as well (see clearS3WriteChecksumFields).
const awsChecksumSetupInputContextID = "AWSChecksum:SetupInputContext"

// clearS3WriteChecksumFields zeroes every input field the S3 serializers bind to
// a forbidden checksum header on the object write shapes, and reports whether
// the input was one of those shapes.
//
// The field sets differ per shape and are taken from the s3 v1.106.0 input
// structs and their awsRestxml_serializeOpHttpBindings*Input serializers:
// PutObjectInput and UploadPartInput carry ChecksumAlgorithm (bound to
// x-amz-sdk-checksum-algorithm) plus the precomputed digests, but no
// ChecksumType; CreateMultipartUploadInput carries only ChecksumAlgorithm (bound
// to x-amz-checksum-algorithm) and ChecksumType; CompleteMultipartUploadInput
// carries the precomputed digests and ChecksumType but no ChecksumAlgorithm.
//
// Per-part checksums inside CompleteMultipartUploadInput.MultipartUpload.Parts
// are deliberately left alone: awsRestxml_serializeDocumentCompletedPart writes
// them as XML elements in the request body, not as headers, so they are a
// legitimate part of the completion request and clearing them could invalidate
// it.
//
// Note that this discards caller-supplied checksums silently: in compat mode a
// precomputed digest passed on an object write (for example via s3blob's
// BeforeWrite hook) is dropped rather than sent. Failing loudly instead is not
// an option here, because by the time this middleware runs there is no way to
// tell a value the caller set deliberately from one the SDK or the S3 transfer
// manager injected on its own; erroring out would break ordinary writes. The
// endpoint would reject the header anyway, so dropping it is what makes the
// write succeed.
//
// It also reports whether the input was one of the object write shapes, so
// callers that need that classification do not duplicate the type switch.
func clearS3WriteChecksumFields(params any) bool {
	switch v := params.(type) {
	case *s3.PutObjectInput:
		v.ChecksumAlgorithm = ""
		v.ChecksumCRC32, v.ChecksumCRC32C, v.ChecksumCRC64NVME = nil, nil, nil
		v.ChecksumMD5, v.ChecksumSHA1, v.ChecksumSHA256, v.ChecksumSHA512 = nil, nil, nil, nil
		v.ChecksumXXHASH128, v.ChecksumXXHASH3, v.ChecksumXXHASH64 = nil, nil, nil
	case *s3.UploadPartInput:
		v.ChecksumAlgorithm = ""
		v.ChecksumCRC32, v.ChecksumCRC32C, v.ChecksumCRC64NVME = nil, nil, nil
		v.ChecksumMD5, v.ChecksumSHA1, v.ChecksumSHA256, v.ChecksumSHA512 = nil, nil, nil, nil
		v.ChecksumXXHASH128, v.ChecksumXXHASH3, v.ChecksumXXHASH64 = nil, nil, nil
	case *s3.CreateMultipartUploadInput:
		v.ChecksumAlgorithm = ""
		v.ChecksumType = ""
	case *s3.CompleteMultipartUploadInput:
		v.ChecksumType = ""
		v.ChecksumCRC32, v.ChecksumCRC32C, v.ChecksumCRC64NVME = nil, nil, nil
		v.ChecksumMD5, v.ChecksumSHA1, v.ChecksumSHA256, v.ChecksumSHA512 = nil, nil, nil, nil
		v.ChecksumXXHASH128, v.ChecksumXXHASH3, v.ChecksumXXHASH64 = nil, nil, nil
	default:
		return false
	}

	return true
}

// suppressS3WriteChecksumSetup replaces the SDK's checksum setup middleware for
// object write operations. It keeps that middleware's ID and stack position, so
// for every other operation (including ones S3 requires a checksum for, such as
// DeleteObjects) the original behavior is delegated to unchanged.
//
// Setting s3.Options.RequestChecksumCalculation to WhenRequired is not enough on
// its own: the S3 transfer manager overrides that option per call with its own
// Options.RequestChecksumCalculation, which defaults to WhenSupported, so the
// SDK re-adds a default CRC32 checksum even in compat mode.
type suppressS3WriteChecksumSetup struct {
	original smithymiddleware.InitializeMiddleware
}

func (m *suppressS3WriteChecksumSetup) ID() string {
	return awsChecksumSetupInputContextID
}

func (m *suppressS3WriteChecksumSetup) HandleInitialize(
	ctx context.Context, in smithymiddleware.InitializeInput, next smithymiddleware.InitializeHandler,
) (smithymiddleware.InitializeOutput, smithymiddleware.Metadata, error) {
	// clearS3WriteChecksumFields both classifies the input and clears it, which
	// keeps the list of object write shapes in one place. Calling it here as well
	// as from the clearing middleware is harmless: it only zeroes fields, so it is
	// idempotent no matter which of the two runs first.
	if clearS3WriteChecksumFields(in.Parameters) {
		// Skip the SDK setup entirely: with no algorithm on the context, the
		// compute middleware is a no-op and the request carries no SDK-computed
		// checksum.
		return next.HandleInitialize(ctx, in)
	}

	if m.original != nil {
		return m.original.HandleInitialize(ctx, in, next)
	}

	return next.HandleInitialize(ctx, in)
}

func stripS3InputChecksumAlgorithm(stack *smithymiddleware.Stack) error {
	// Clear caller-supplied checksum fields for every operation. This cannot be
	// folded into the swap below: CreateMultipartUpload and CompleteMultipartUpload
	// register no checksum setup middleware at all in s3 v1.106.0, so the swap
	// never happens for them, yet both bind checksum inputs to headers.
	clearInput := smithymiddleware.InitializeMiddlewareFunc(
		"iceberg-go/strip-s3-input-checksum-algorithm",
		func(ctx context.Context, in smithymiddleware.InitializeInput, next smithymiddleware.InitializeHandler) (smithymiddleware.InitializeOutput, smithymiddleware.Metadata, error) {
			clearS3WriteChecksumFields(in.Parameters)

			return next.HandleInitialize(ctx, in)
		},
	)
	if err := stack.Initialize.Add(clearInput, smithymiddleware.Before); err != nil {
		return err
	}

	// Additionally suppress the SDK's own computed checksum on write operations
	// that do set up a checksum context (PutObject, UploadPart). A missing
	// middleware is expected for the operations noted above, so it is not an error.
	suppress := &suppressS3WriteChecksumSetup{}
	if original, err := stack.Initialize.Swap(awsChecksumSetupInputContextID, suppress); err == nil {
		suppress.original = original
	}

	return nil
}

// SDK-internal headers GCS's S3 interop endpoint rejects in the SigV4 signed set; removing
// them before signing (all ops, reads included) avoids SignatureDoesNotMatch.
var gcsIncompatibleSignedHeaders = []string{
	"Amz-Sdk-Invocation-Id",
	"Amz-Sdk-Request",
	"Accept-Encoding",
}

// stripGCSIncompatibleSignedHeaders removes those headers before signing, then re-adds
// Accept-Encoding: identity after signing so it's on the wire but unsigned (no gzip, GCS-safe).
func stripGCSIncompatibleSignedHeaders(stack *smithymiddleware.Stack) error {
	strip := smithymiddleware.FinalizeMiddlewareFunc(
		"iceberg-go/strip-gcs-incompatible-signed-headers",
		func(ctx context.Context, in smithymiddleware.FinalizeInput, next smithymiddleware.FinalizeHandler) (smithymiddleware.FinalizeOutput, smithymiddleware.Metadata, error) {
			if req, ok := in.Request.(*smithyhttp.Request); ok {
				for _, h := range gcsIncompatibleSignedHeaders {
					req.Header.Del(h)
				}
			}

			return next.HandleFinalize(ctx, in)
		},
	)

	restore := smithymiddleware.FinalizeMiddlewareFunc(
		"iceberg-go/restore-accept-encoding-identity",
		func(ctx context.Context, in smithymiddleware.FinalizeInput, next smithymiddleware.FinalizeHandler) (smithymiddleware.FinalizeOutput, smithymiddleware.Metadata, error) {
			if req, ok := in.Request.(*smithyhttp.Request); ok {
				req.Header.Set("Accept-Encoding", "identity")
			}

			return next.HandleFinalize(ctx, in)
		},
	)

	if err := stack.Finalize.Insert(strip, "Signing", smithymiddleware.Before); err != nil {
		if err := stack.Finalize.Add(strip, smithymiddleware.Before); err != nil {
			return err
		}
	}
	if err := stack.Finalize.Insert(restore, "Signing", smithymiddleware.After); err != nil {
		if err := stack.Finalize.Add(restore, smithymiddleware.After); err != nil {
			return err
		}
	}

	return nil
}

func s3CompatModeEnabled(props map[string]string) bool {
	if v, ok := props[io.S3CompatMode]; ok {
		if enabled, err := strconv.ParseBool(v); err == nil {
			return enabled
		}
	}

	return false
}
