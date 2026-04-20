// Package wsproto provides helpers for reading and writing Proto messages.
package wsproto

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
	"sync"

	"github.com/coder/websocket"
)

var bpool sync.Pool

// Get returns a buffer from the pool or creates a new one if
// the pool is empty.
func getBuffer() *bytes.Buffer {
	b := bpool.Get()
	if b == nil {
		return &bytes.Buffer{}
	}
	return b.(*bytes.Buffer)
}

// Put returns a buffer into the pool.
func putBuffer(b *bytes.Buffer) {
	b.Reset()
	bpool.Put(b)
}

func Read(ctx context.Context, c *websocket.Conn, v proto.Message) error {
	_, r, err := c.Reader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get websocket reader: %w", err)
	}

	b := getBuffer()
	defer putBuffer(b)

	_, err = b.ReadFrom(r)
	if err != nil {
		return err
	}

	err = proto.Unmarshal(b.Bytes(), v)
	if err != nil {
		_ = c.Close(websocket.StatusInvalidFramePayloadData, "failed to unmarshal Protobuf")
		return fmt.Errorf("failed to unmarshal Protobuf: %w", err)
	}

	return nil
}

func Write(ctx context.Context, c *websocket.Conn, v proto.Message) error {
	marshaled, err := proto.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal Protobuf message: %w", err)
	}

	err = c.Write(ctx, websocket.MessageBinary, marshaled)
	if err != nil {
		return fmt.Errorf("could not write message to websocket: %w", err)
	}

	return nil
}
