package eventstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"

	"github.com/inngest/inngest/pkg/consts"
)

var (
	ErrInvalidRequestBody = fmt.Errorf("Request body must contain an event object or an array of event objects")
	ErrEventTooLarge      = fmt.Errorf("Event is over the max size")
)

type StreamItem struct {
	N    int
	Item json.RawMessage
}

// ParseStream parses a reader, publishing a stream of JSON-encoded events to the given channel,
// ensuring that no individual event is too large.
//
// This closes the given channel once the JSON stream in the given reader has been parsed.
// Supports both JSON and multipart/form-data content types.
//
// Usage:
//
//			var err error
//			go func() {
//			        err = ParseStream(ctx, r, stream, contentType)
//			()
//
//			for bytes := range stream {
//			        // consume event, transform event, etc
//			}
//
//	     if err != nil {
//	             // handle error
//	     }
func ParseStream(
	ctx context.Context,
	r io.Reader,
	stream chan StreamItem,
	maxSize int,
	contentType string,
) error {
	defer func() {
		close(stream)
	}()

	// Check if content type is multipart/form-data
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType == "multipart/form-data" {
		return parseMultipartStream(ctx, r, stream, maxSize, params["boundary"])
	}

	// Default to JSON parsing
	d := json.NewDecoder(r)

	token, err := d.Token()
	if err == io.EOF {
		return nil
	}

	delim, ok := token.(json.Delim)
	if !ok {
		// Invalid type
		return ErrInvalidRequestBody
	}

	switch delim {
	case '{':
		// We've already peeked the first char.  Read all, then prepend a '{'
		byt, err := io.ReadAll(d.Buffered())
		if err != nil {
			return err
		}
		// d.Buffered() only returns a portion of the buffered stream;  read the rest
		// and concat.
		extra, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		data := append([]byte("{"), byt...)
		data = append(data, extra...)
		if len(data) > maxSize {
			return fmt.Errorf("%w: Max %d bytes / Size %d bytes", ErrEventTooLarge, maxSize, len(data))
		}

		select {
		case stream <- StreamItem{Item: data}:
			// Sent
		case <-ctx.Done():
			// Early exit; a problem somewhere else in the pipeline
			return nil
		}
	case '[':
		i := 0
		// Parse a stream of tokens
		for d.More() {
			if i == consts.MaxEvents {
				return &ErrEventCount{Max: consts.MaxEvents}
			}

			jsonEvt := json.RawMessage{}
			if err := d.Decode(&jsonEvt); err != nil {
				return err
			}
			if len(jsonEvt) > maxSize {
				return fmt.Errorf("%w: Max %d bytes / Size %d bytes", ErrEventTooLarge, maxSize, len(jsonEvt))
			}
			select {
			case stream <- StreamItem{N: i, Item: jsonEvt}:
				// Sent
				i++
			case <-ctx.Done():
				// Early exit; a problem somewhere else in the pipeline
				return nil
			}
		}
	default:
		return ErrInvalidRequestBody
	}
	return nil
}

// parseMultipartStream parses multipart/form-data and extracts JSON events from
// form fields
func parseMultipartStream(
	ctx context.Context,
	r io.Reader,
	stream chan StreamItem,
	maxSize int,
	boundary string,
) error {
	reader := multipart.NewReader(r, boundary)

	// Collect all form fields
	formData := make(map[string]string)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Read the part data
		data, err := io.ReadAll(part)
		part.Close()
		if err != nil {
			return err
		}

		// Skip empty parts
		if len(data) == 0 {
			continue
		}

		fieldName := part.FormName()
		if fieldName != "" {
			formData[fieldName] = string(data)
		}
	}

	if len(formData) > 0 {
		byt, err := json.Marshal(formData)
		if err != nil {
			return err
		}

		if len(byt) > maxSize {
			return fmt.Errorf("%w: Max %d bytes / Size %d bytes", ErrEventTooLarge, maxSize, len(byt))
		}

		select {
		case stream <- StreamItem{N: 0, Item: json.RawMessage(byt)}:
			// Success
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}
