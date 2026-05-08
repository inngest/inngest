package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "inngest.conformance.yaml")
	err := os.WriteFile(path, []byte(`
transport: serve
suites:
  - core
report:
  format: json
fixtures:
  root: ./fixtures
golden:
  root: ./golden
`), 0o600)
	require.NoError(t, err)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, TransportServe, cfg.Transport)
	require.Equal(t, []string{"core"}, cfg.Suites)
	require.Equal(t, ReportFormatJSON, cfg.Report.Format)
	require.Equal(t, "./fixtures", cfg.Fixtures.Root)
	require.Equal(t, "./golden", cfg.Golden.Root)
	require.Equal(t, GoldenModeSemantic, cfg.Golden.Mode)
}

func TestLoadConfigRejectsInvalidTransport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "inngest.conformance.json")
	err := os.WriteFile(path, []byte(`{"transport":"smtp"}`), 0o600)
	require.NoError(t, err)

	_, err = LoadConfig(path)
	require.ErrorContains(t, err, `unknown transport "smtp"`)
}
