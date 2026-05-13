package grafanalgtm

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	GrafanaPort    = "3000/tcp"
	LokiPort       = "3100/tcp"
	TempoPort      = "3200/tcp"
	OtlpGrpcPort   = "4317/tcp"
	OtlpHttpPort   = "4318/tcp" //nolint:revive,staticcheck //FIXME
	PrometheusPort = "9090/tcp"
)

// GrafanaLGTMContainer represents the Grafana LGTM container type used in the module
type GrafanaLGTMContainer struct {
	testcontainers.Container
}

// Run creates an instance of the Grafana LGTM container type
func Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*GrafanaLGTMContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        img,
		ExposedPorts: []string{GrafanaPort, LokiPort, TempoPort, OtlpGrpcPort, OtlpHttpPort, PrometheusPort},
		WaitingFor:   wait.ForLog(".*The OpenTelemetry collector and the Grafana LGTM stack are up and running.*\\s").AsRegexp().WithOccurrence(1),
	}

	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&genericContainerReq); err != nil {
			return nil, fmt.Errorf("customize: %w", err)
		}
	}

	container, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	if err != nil {
		return nil, fmt.Errorf("generic container: %w", err)
	}

	c := &GrafanaLGTMContainer{Container: container}

	url, err := c.OtlpHttpEndpoint(ctx)
	if err != nil {
		// return the container instance to allow the caller to clean up
		return c, fmt.Errorf("otlp http endpoint: %w", err)
	}

	log.Printf("Access to the Grafana dashboard: %s", url)

	return c, nil
}

// WithAdminCredentials sets the admin credentials for the Grafana LGTM container
func WithAdminCredentials(user, password string) testcontainers.ContainerCustomizer {
	return testcontainers.WithEnv(map[string]string{
		"GF_SECURITY_ADMIN_USER":     user,
		"GF_SECURITY_ADMIN_PASSWORD": password,
	})
}

// LokiEndpoint returns the Loki endpoint
func (c *GrafanaLGTMContainer) LokiEndpoint(ctx context.Context) (string, error) {
	url, err := c.PortEndpoint(ctx, LokiPort, "")
	if err != nil {
		return "", err
	}

	return url, nil
}

// MustLokiEndpoint returns the Loki endpoint or panics if an error occurs
func (c *GrafanaLGTMContainer) MustLokiEndpoint(ctx context.Context) string {
	url, err := c.LokiEndpoint(ctx)
	if err != nil {
		panic(err)
	}

	return url
}

// TempoEndpoint returns the Tempo endpoint
func (c *GrafanaLGTMContainer) TempoEndpoint(ctx context.Context) (string, error) {
	url, err := c.PortEndpoint(ctx, TempoPort, "")
	if err != nil {
		return "", err
	}

	return url, nil
}

// MustTempoEndpoint returns the Tempo endpoint or panics if an error occurs
func (c *GrafanaLGTMContainer) MustTempoEndpoint(ctx context.Context) string {
	url, err := c.TempoEndpoint(ctx)
	if err != nil {
		panic(err)
	}

	return url
}

// HttpEndpoint returns the HTTP URL
//
//nolint:revive,staticcheck //FIXME
func (c *GrafanaLGTMContainer) HttpEndpoint(ctx context.Context) (string, error) {
	url, err := c.PortEndpoint(ctx, GrafanaPort, "")
	if err != nil {
		return "", err
	}

	return url, nil
}

// MustHttpEndpoint returns the HTTP endpoint or panics if an error occurs
//
//nolint:revive,staticcheck //FIXME
func (c *GrafanaLGTMContainer) MustHttpEndpoint(ctx context.Context) string {
	url, err := c.HttpEndpoint(ctx)
	if err != nil {
		panic(err)
	}

	return url
}

// OtlpHttpEndpoint returns the OTLP HTTP endpoint
//
//nolint:revive,staticcheck //FIXME
func (c *GrafanaLGTMContainer) OtlpHttpEndpoint(ctx context.Context) (string, error) {
	url, err := c.PortEndpoint(ctx, OtlpHttpPort, "")
	if err != nil {
		return "", err
	}

	return url, nil
}

// MustOtlpHttpEndpoint returns the OTLP HTTP endpoint or panics if an error occurs
//
//nolint:revive,staticcheck //FIXME
func (c *GrafanaLGTMContainer) MustOtlpHttpEndpoint(ctx context.Context) string {
	url, err := c.OtlpHttpEndpoint(ctx)
	if err != nil {
		panic(err)
	}

	return url
}

// OtlpGrpcEndpoint returns the OTLP gRPC endpoint
func (c *GrafanaLGTMContainer) OtlpGrpcEndpoint(ctx context.Context) (string, error) {
	url, err := c.PortEndpoint(ctx, OtlpGrpcPort, "")
	if err != nil {
		return "", err
	}

	return url, nil
}

// MustOtlpGrpcEndpoint returns the OTLP gRPC endpoint or panics if an error occurs
func (c *GrafanaLGTMContainer) MustOtlpGrpcEndpoint(ctx context.Context) string {
	url, err := c.OtlpGrpcEndpoint(ctx)
	if err != nil {
		panic(err)
	}

	return url
}

// PrometheusHttpEndpoint returns the Prometheus HTTP endpoint
//
//nolint:revive,staticcheck //FIXME
func (c *GrafanaLGTMContainer) PrometheusHttpEndpoint(ctx context.Context) (string, error) {
	url, err := c.PortEndpoint(ctx, PrometheusPort, "")
	if err != nil {
		return "", err
	}

	return url, nil
}

// MustPrometheusHttpEndpoint returns the Prometheus HTTP endpoint or panics if an error occurs
//
//nolint:revive,staticcheck //FIXME
func (c *GrafanaLGTMContainer) MustPrometheusHttpEndpoint(ctx context.Context) string {
	url, err := c.PrometheusHttpEndpoint(ctx)
	if err != nil {
		panic(err)
	}

	return url
}
