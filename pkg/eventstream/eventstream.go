package eventstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"

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
// Supports JSON, multipart/form-data, and application/x-www-form-urlencoded content types.
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

	// Default to JSON parsing
	d := json.NewDecoder(r)

	token, err := d.Token()
	if err == io.EOF {
		return nil
	}

	delim, ok := token.(json.Delim)
	if !ok {
		if contentType != "application/json" {
			// We get here if the first token is not a JSON delimiter and the
			// request's content-type is not explicitly JSON. This means we may
			// be dealing with a non-JSON request that we need to parse in a
			// different way

			// Reconstruct the full body by combining the decoder's buffer with
			// the original reader. We can assume that the decoder buffer has 0
			// consumed bytes because `Decoder.Token()` does not consume bytes
			// if the returned value would be invalid JSON
			r := io.MultiReader(d.Buffered(), r)

			mediaType, params, _ := mime.ParseMediaType(contentType)
			switch mediaType {
			case "multipart/form-data":
				return parseMultipartStream(ctx, r, stream, maxSize, params["boundary"])
			case "application/x-www-form-urlencoded":
				return parseFormUrlencodedStream(ctx, r, stream, maxSize)
			}
		}
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
	formData := make(map[string][]string)

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

		fieldName := part.FormName()
		if fieldName != "" {
			formData[fieldName] = append(formData[fieldName], string(data))
		}
	}

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

	return nil
}

// parseFormUrlencodedStream parses application/x-www-form-urlencoded data and
// extracts form fields as a JSON event
func parseFormUrlencodedStream(
	ctx context.Context,
	r io.Reader,
	stream chan StreamItem,
	maxSize int,
) error {
	// Read all data from the reader
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Parse the form data
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}

	byt, err := json.Marshal(values)
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

	return nil
}
