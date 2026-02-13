package extractors

import (
	"github.com/inngest/go-httpstat"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

//tygo:generate
const (
	KindInngestHTTPTiming metadata.Kind = "inngest.http.timing"
)

// HTTPTimingMetadata contains detailed HTTP connection timing phases
// from httpstat, capturing the duration of each phase in the HTTP
// request lifecycle.
//
//tygo:generate
type HTTPTimingMetadata struct {
	// DNSLookupMs is the time spent resolving the domain name.
	DNSLookupMs int64 `json:"dns_lookup_ms"`
	// TCPConnectionMs is the time spent establishing the TCP connection.
	TCPConnectionMs int64 `json:"tcp_connection_ms"`
	// TLSHandshakeMs is the time spent on TLS negotiation.
	TLSHandshakeMs int64 `json:"tls_handshake_ms"`
	// ServerProcessingMs is the time from request sent to first byte received (TTFB).
	ServerProcessingMs int64 `json:"server_processing_ms"`
	// ContentTransferMs is the time spent downloading the response body.
	ContentTransferMs int64 `json:"content_transfer_ms"`
	// TotalMs is the total request duration.
	TotalMs int64 `json:"total_ms"`
}

func (m HTTPTimingMetadata) Kind() metadata.Kind {
	return KindInngestHTTPTiming
}

func (m HTTPTimingMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (m HTTPTimingMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(m)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

// ExtractHTTPTimingMetadata converts an httpstat.Result into structured
// metadata containing the timing breakdown of each HTTP request phase.
func ExtractHTTPTimingMetadata(stat *httpstat.Result) metadata.Structured {
	contentTransferMs := int64(0)
	if stat.Total > stat.StartTransfer {
		contentTransferMs = (stat.Total - stat.StartTransfer).Milliseconds()
	}

	return &HTTPTimingMetadata{
		DNSLookupMs:        stat.DNSLookup.Milliseconds(),
		TCPConnectionMs:    stat.TCPConnection.Milliseconds(),
		TLSHandshakeMs:     stat.TLSHandshake.Milliseconds(),
		ServerProcessingMs: stat.ServerProcessing.Milliseconds(),
		ContentTransferMs:  contentTransferMs,
		TotalMs:            stat.Total.Milliseconds(),
	}
}
