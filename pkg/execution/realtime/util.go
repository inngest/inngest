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

// TeeStreamReaderToAPI buffers the reader then publishes the data to the HTTP
// API as a side-effect. The returned reader always contains the full body, even
// if publishing fails.
func TeeStreamReaderToAPI(reader io.Reader, publishURL string, opts TeeStreamOptions) (io.Reader, error) {
	if opts.Channel == "" || opts.Topic == "" || opts.Token == "" {
		// Bypass, don't do anything.
		return reader, nil
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = publishToAPI(data, publishURL, opts)
	return bytes.NewReader(data), err
}

func publishToAPI(data []byte, publishURL string, opts TeeStreamOptions) error {
	qp := url.Values{}
	qp.Add("channel", opts.Channel)
	qp.Add("topic", opts.Topic)

	if len(opts.Metadata) > 0 {
		byt, _ := json.Marshal(opts.Metadata)
		qp.Add("metadata", string(byt))
	}

	req, err := http.NewRequest(http.MethodPost, publishURL+"?"+qp.Encode(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "text/stream")
	req.Header.Add("Authorization", opts.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid status code publishing stream: %d", resp.StatusCode)
	}

	return nil
}
