package realtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type TeeStreamOptions struct {
	Channel  string
	Topic    string
	Token    string
	Metadata map[string]any
}

// TeeStreamReaderToAPI is a utility function that publishes a reader to the HTTP API,
// streamed.  It returns a reader which contains the data read from the original reader.
func TeeStreamReaderToAPI(reader io.Reader, publishURL string, opts TeeStreamOptions) (io.Reader, error) {
	if opts.Channel == "" || opts.Topic == "" || opts.Token == "" {
		// Bypass, don't do anything.
		return reader, nil
	}

	buf := bytes.NewBuffer(nil)
	tee := io.TeeReader(reader, buf)

	qp := url.Values{}
	qp.Add("channel", opts.Channel)
	qp.Add("topic", opts.Topic)

	if len(opts.Metadata) > 0 {
		byt, _ := json.Marshal(opts.Metadata)
		qp.Add("metadata", string(byt))
	}

	// This pushes the request directly to the API,
	req, err := http.NewRequest(http.MethodPost, publishURL+"?"+qp.Encode(), tee)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "text/stream")
	req.Header.Add("Authorization", opts.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("invalid status code publishing stream: %d", resp.StatusCode)
	}

	return buf, nil
}
