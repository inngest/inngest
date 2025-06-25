// Package httpstat traces HTTP latency infomation (DNSLookup, TCP Connection and so on) on any golang HTTP request.
// It uses `httptrace` package. Just create `go-httpstat` powered `context.Context` and give it your `http.Request` (no big code modification is required).
package httpstat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// Result stores httpstat info.
type Result struct {
	mutex sync.Mutex

	// The following are duration for each phase
	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	contentTransfer  time.Duration

	// DNS and connection Info
	IsIPv6      bool
	Addresses   []net.IPAddr
	ConnectedTo ConnectedTo

	// The followings are timeline of request
	NameLookup    time.Duration
	Connect       time.Duration
	Pretransfer   time.Duration
	StartTransfer time.Duration
	Total         time.Duration

	// HostID is an optional host ID that made this request.
	HostID string

	t5 time.Time // need to be provided from outside

	dnsStart      time.Time
	dnsDone       time.Time
	tcpStart      time.Time
	tcpDone       time.Time
	tlsStart      time.Time
	tlsDone       time.Time
	serverStart   time.Time
	serverDone    time.Time
	transferStart time.Time
	transferDone  time.Time // need to be provided from outside

	// isTLS is true when connection seems to use TLS
	isTLS bool

	// isReused is true when connection is reused (keep-alive)
	isReused bool
}

type ConnectedTo struct {
	IP   string
	Port int
	Zone string
}

func (r *Result) durations() map[string]time.Duration {
	return map[string]time.Duration{
		"DNSLookup":        r.DNSLookup,
		"TCPConnection":    r.TCPConnection,
		"TLSHandshake":     r.TLSHandshake,
		"ServerProcessing": r.ServerProcessing,
		"ContentTransfer":  r.contentTransfer,

		"NameLookup":    r.NameLookup,
		"Connect":       r.Connect,
		"Pretransfer":   r.Connect,
		"StartTransfer": r.StartTransfer,
		"Total":         r.Total,
	}
}

// Format formats stats result.
func (r *Result) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "DNS lookup:        %4d ms\n",
				int(r.DNSLookup/time.Millisecond))
			fmt.Fprintf(&buf, "TCP connection:    %4d ms\n",
				int(r.TCPConnection/time.Millisecond))
			fmt.Fprintf(&buf, "TLS handshake:     %4d ms\n",
				int(r.TLSHandshake/time.Millisecond))
			fmt.Fprintf(&buf, "Server processing: %4d ms\n",
				int(r.ServerProcessing/time.Millisecond))

			if r.Total > 0 {
				fmt.Fprintf(&buf, "Content transfer:  %4d ms\n\n",
					int(r.contentTransfer/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Content transfer:  %4s ms\n\n", "-")
			}

			fmt.Fprintf(&buf, "Name Lookup:    %4d ms\n",
				int(r.NameLookup/time.Millisecond))
			fmt.Fprintf(&buf, "Connect:        %4d ms\n",
				int(r.Connect/time.Millisecond))
			fmt.Fprintf(&buf, "Pre Transfer:   %4d ms\n",
				int(r.Pretransfer/time.Millisecond))
			fmt.Fprintf(&buf, "Start Transfer: %4d ms\n",
				int(r.StartTransfer/time.Millisecond))

			if r.Total > 0 {
				fmt.Fprintf(&buf, "Total:          %4d ms\n",
					int(r.Total/time.Millisecond))
			} else {
				fmt.Fprintf(&buf, "Total:          %4s ms\n", "-")
			}
			io.WriteString(s, buf.String())
			return
		}

		fallthrough
	case 's', 'q':
		d := r.durations()
		list := make([]string, 0, len(d))
		for k, v := range d {
			// Handle when End function is not called
			if (k == "ContentTransfer" || k == "Total") && r.t5.IsZero() {
				list = append(list, fmt.Sprintf("%s: - ms", k))
				continue
			}
			list = append(list, fmt.Sprintf("%s: %d ms", k, v/time.Millisecond))
		}
		io.WriteString(s, strings.Join(list, ", "))
	}
}

// WithHTTPStat is a wrapper of httptrace.WithClientTrace. It records the
// time of each httptrace hooks.
func WithHTTPStat(ctx context.Context, r *Result) context.Context {
	return withClientTrace(ctx, r)
}
