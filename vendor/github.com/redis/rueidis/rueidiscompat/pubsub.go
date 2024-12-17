// Copyright (c) 2013 The github.com/go-redis/redis Authors.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package rueidiscompat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/rueidis"
)

type PubSub interface {
	Close() error
	Subscribe(ctx context.Context, channels ...string) error
	PSubscribe(ctx context.Context, patterns ...string) error
	SSubscribe(ctx context.Context, channels ...string) error
	Unsubscribe(ctx context.Context, channels ...string) error
	PUnsubscribe(ctx context.Context, patterns ...string) error
	SUnsubscribe(ctx context.Context, channels ...string) error
	Ping(ctx context.Context, payload ...string) error
	ReceiveTimeout(ctx context.Context, timeout time.Duration) (any, error)
	Receive(ctx context.Context) (any, error)
	ReceiveMessage(ctx context.Context) (*Message, error)
	Channel(opts ...ChannelOption) <-chan *Message
	ChannelWithSubscriptions(opts ...ChannelOption) <-chan any
	String() string
}

type ChannelOption func(c *chopt)

type chopt struct {
	chanSize int
}

// WithChannelSize specifies the Go chan size that is used to buffer incoming messages.
// The default is 1000 messages.
func WithChannelSize(size int) ChannelOption {
	return func(c *chopt) {
		c.chanSize = size
	}
}

// WithChannelHealthCheckInterval is an empty ChannelOption to keep compatibility
func WithChannelHealthCheckInterval(_ time.Duration) ChannelOption {
	return func(c *chopt) {}
}

// WithChannelSendTimeout is an empty ChannelOption to keep compatibility
func WithChannelSendTimeout(_ time.Duration) ChannelOption {
	return func(c *chopt) {}
}

// Subscription received after a successful subscription to channel.
type Subscription struct {
	// Can be "subscribe", "unsubscribe", "psubscribe" or "punsubscribe".
	Kind string
	// Channel name we have subscribed to.
	Channel string
	// Number of channels we are currently subscribed to.
	Count int
}

func (m *Subscription) String() string {
	return fmt.Sprintf("%s: %s", m.Kind, m.Channel)
}

// Message received as result of a PUBLISH command issued by another client.
type Message struct {
	Channel      string
	Pattern      string
	Payload      string
	PayloadSlice []string
}

func (m *Message) String() string {
	return fmt.Sprintf("Message<%s: %s>", m.Channel, m.Payload)
}

func newPubSub(client rueidis.Client) *pubsub {
	return &pubsub{
		rc:        client,
		channels:  make(map[string]bool),
		patterns:  make(map[string]bool),
		schannels: make(map[string]bool),
	}
}

type pubsub struct {
	mu sync.Mutex

	rc      rueidis.Client
	mc      rueidis.DedicatedClient
	mcancel func()

	channels  map[string]bool
	patterns  map[string]bool
	schannels map[string]bool

	allCh chan any
	msgCh chan *Message
}

func (p *pubsub) mconn() rueidis.DedicatedClient {
	if p.mc == nil {
		p.mc, p.mcancel = p.rc.Dedicate()
	}
	return p.mc
}

func (p *pubsub) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.mcancel != nil {
		p.mc.SetPubSubHooks(rueidis.PubSubHooks{})
		p.mcancel()
		p.mc = nil
		p.mcancel = nil
	}
	p.channels = make(map[string]bool)
	p.patterns = make(map[string]bool)
	p.schannels = make(map[string]bool)
	return nil
}

func (p *pubsub) Subscribe(ctx context.Context, channels ...string) error {
	if len(channels) == 0 {
		return nil
	}

	p.ChannelWithSubscriptions()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, channel := range channels {
		p.channels[channel] = true
	}

	c := p.mconn()
	return c.Do(ctx, c.B().Subscribe().Channel(channels...).Build()).Error()
}

func (p *pubsub) PSubscribe(ctx context.Context, patterns ...string) error {
	if len(patterns) == 0 {
		return nil
	}

	p.ChannelWithSubscriptions()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pattern := range patterns {
		p.patterns[pattern] = true
	}

	c := p.mconn()
	return c.Do(ctx, c.B().Psubscribe().Pattern(patterns...).Build()).Error()
}

func (p *pubsub) SSubscribe(ctx context.Context, channels ...string) error {
	if len(channels) == 0 {
		return nil
	}

	p.ChannelWithSubscriptions()

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.channels) != 0 || len(p.patterns) != 0 {
		return errors.New("pubsub: cannot use SSubscribe after using Subscribe or PSubscribe")
	}

	for _, channel := range channels {
		p.schannels[channel] = true
	}

	c := p.mconn()
	return c.Do(ctx, c.B().Ssubscribe().Channel(channels...).Build()).Error()
}

func (p *pubsub) Unsubscribe(ctx context.Context, channels ...string) error {
	if len(channels) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, channel := range channels {
		delete(p.channels, channel)
	}

	c := p.mconn()
	return c.Do(ctx, c.B().Unsubscribe().Channel(channels...).Build()).Error()
}

func (p *pubsub) PUnsubscribe(ctx context.Context, patterns ...string) error {
	if len(patterns) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pattern := range patterns {
		delete(p.patterns, pattern)
	}

	c := p.mconn()
	return c.Do(ctx, c.B().Punsubscribe().Pattern(patterns...).Build()).Error()
}

func (p *pubsub) SUnsubscribe(ctx context.Context, channels ...string) error {
	if len(channels) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, channel := range channels {
		delete(p.schannels, channel)
	}

	c := p.mconn()
	return c.Do(ctx, c.B().Sunsubscribe().Channel(channels...).Build()).Error()
}

func (p *pubsub) Ping(_ context.Context, _ ...string) error {
	return nil // we already ping the connection periodically by default
}

func (p *pubsub) reset() {
	if p.mcancel != nil {
		p.mcancel()
		p.mc = nil
		p.mcancel = nil
	}
	for channel := range p.channels {
		p.channels[channel] = false
	}
	for pattern := range p.patterns {
		p.patterns[pattern] = false
	}
	for schannel := range p.schannels {
		p.schannels[schannel] = false
	}
}

func (p *pubsub) resubscribe(ctx context.Context) rueidis.DedicatedClient {
	p.mu.Lock()
	defer p.mu.Unlock()
retry:
	c := p.mconn()
	ok := len(p.schannels) == 0 && len(p.channels) == 0 && len(p.patterns) == 0
	if len(p.schannels) != 0 {
		builder := c.B().Ssubscribe().Channel()
		for channel, ok := range p.schannels {
			if !ok {
				builder = builder.Channel(channel)
				p.schannels[channel] = true
			}
		}
		if cmd := builder.Build(); len(cmd.Commands()) > 1 {
			if err := c.Do(ctx, cmd).NonRedisError(); err != nil {
				p.reset()
				goto retry
			}
			ok = true
		}
	}
	if len(p.channels) != 0 {
		builder := c.B().Subscribe().Channel()
		for channel, ok := range p.channels {
			if !ok {
				builder = builder.Channel(channel)
				p.channels[channel] = true
			}
		}
		if cmd := builder.Build(); len(cmd.Commands()) > 1 {
			if err := c.Do(ctx, cmd).NonRedisError(); err != nil {
				p.reset()
				goto retry
			}
			ok = true
		}
	}
	if len(p.patterns) != 0 {
		builder := c.B().Psubscribe().Pattern()
		for pattern, ok := range p.patterns {
			if !ok {
				builder = builder.Pattern(pattern)
				p.patterns[pattern] = true
			}
		}
		if cmd := builder.Build(); len(cmd.Commands()) > 1 {
			if err := c.Do(ctx, cmd).NonRedisError(); err != nil {
				p.reset()
				goto retry
			}
			ok = true
		}
	}
	if !ok {
		if err := c.Do(ctx, c.B().Ping().Build()).NonRedisError(); err != nil {
			p.reset()
			goto retry
		}
	}
	return c
}

func (p *pubsub) ReceiveTimeout(ctx context.Context, timeout time.Duration) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case m := <-p.ChannelWithSubscriptions():
		if m == nil {
			return nil, errors.New("redis: client is closed")
		}
		return m, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *pubsub) Receive(_ context.Context) (any, error) {
	m := <-p.ChannelWithSubscriptions()
	if m == nil {
		return nil, errors.New("redis: client is closed")
	}
	return m, nil
}

func (p *pubsub) ReceiveMessage(_ context.Context) (*Message, error) {
	m := <-p.Channel()
	if m == nil {
		return nil, errors.New("redis: client is closed")
	}
	return m, nil
}

func (p *pubsub) Channel(opts ...ChannelOption) <-chan *Message {
	ch := p.ChannelWithSubscriptions(opts...)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.msgCh != nil {
		return p.msgCh
	}
	msgCh := make(chan *Message)
	p.msgCh = msgCh
	go func() {
		for m := range ch {
			if msg, ok := m.(*Message); ok {
				msgCh <- msg
			}
		}
		p.mu.Lock()
		if p.msgCh == msgCh {
			p.msgCh = nil
		}
		p.mu.Unlock()
		close(msgCh)
	}()
	return msgCh
}

func (p *pubsub) ChannelWithSubscriptions(opts ...ChannelOption) <-chan any {
	p.mu.Lock()
	if p.allCh != nil {
		p.mu.Unlock()
		return p.allCh
	}
	opt := &chopt{chanSize: 1000}
	for _, fn := range opts {
		fn(opt)
	}
	allCh := make(chan any, opt.chanSize)
	p.allCh = allCh
	p.mu.Unlock()

	resubscribe := func() <-chan error {
		c := p.resubscribe(context.Background())
		return c.SetPubSubHooks(rueidis.PubSubHooks{
			OnMessage: func(m rueidis.PubSubMessage) {
				msg := &Message{
					Channel: m.Channel,
					Pattern: m.Pattern,
					Payload: m.Message,
				}
				select {
				case allCh <- msg:
				default:
				}
			},
			OnSubscription: func(s rueidis.PubSubSubscription) {
				sub := &Subscription{
					Kind:    s.Kind,
					Channel: s.Channel,
					Count:   int(s.Count),
				}
				select {
				case allCh <- sub:
				default:
				}
			},
		})
	}
	go func(wait <-chan error) {
		for {
			if err := <-wait; err == nil {
				p.mu.Lock()
				if p.allCh == allCh {
					p.allCh = nil
				}
				p.mu.Unlock()
				close(allCh)
				return
			}
			p.mu.Lock()
			p.reset()
			p.mu.Unlock()
			wait = resubscribe()
		}
	}(resubscribe())

	return allCh
}

func (p *pubsub) String() string {
	channels := mapKeys(p.channels)
	channels = append(channels, mapKeys(p.patterns)...)
	channels = append(channels, mapKeys(p.schannels)...)
	return fmt.Sprintf("PubSub(%s)", strings.Join(channels, ", "))
}

func mapKeys(m map[string]bool) []string {
	s := make([]string, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	return s
}
