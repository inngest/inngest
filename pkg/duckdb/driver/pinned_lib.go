package driver

// pinnedLibChecksums are the SHA256 digests of each pinned-version DuckDB
// shared library release archive EnsureLibrary might download, keyed by asset
// name. DuckDB v2.0.0 publishes shared libraries as duckdb-shared-libs-<dist>.tar.gz.
// Digests were verified against the official GitHub v2.0.0 release metadata so
// EnsureLibrary fails closed on corruption or tampering in transit.
var pinnedLibChecksums = map[string]string{
	"duckdb-shared-libs-osx-universal.tar.gz": "14cd63180614cfd961f7959146ecd9087527ff20b67311efdc3fa1c29c18204d",
	"duckdb-shared-libs-linux-amd64.tar.gz":   "993ad89e7dfca66cd6da994c2cad090c2fc160ddf9f550498a098ab4bb13cfb1",
	"duckdb-shared-libs-linux-arm64.tar.gz":   "c60eecf91143ca6bbac55c4f4d40dc22520488b0ac257c86e0fe9a2b451cf2b7",
	"duckdb-shared-libs-windows-amd64.tar.gz": "2b83169128239de5f45fe21de80f99bf436e53aa4bb24020dd951a1ba8d4b95f",
}
