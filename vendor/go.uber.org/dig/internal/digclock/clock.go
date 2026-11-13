// Copyright (c) 2024 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package digclock

import (
	"time"
)

// Clock defines how dig accesses time.
type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

// System is the default implementation of Clock based on real time.
var System Clock = systemClock{}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (systemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Mock is a fake source of time.
// It implements standard time operations, but allows
// the user to control the passage of time.
//
// Use the [Mock.Add] method to progress time.
//
// Note that this implementation is not safe for concurrent use.
type Mock struct {
	now time.Time
}

var _ Clock = (*Mock)(nil)

// NewMock creates a new mock clock with the current time set to the current time.
func NewMock() *Mock {
	return &Mock{now: time.Now()}
}

// Now returns the current time.
func (m *Mock) Now() time.Time {
	return m.now
}

// Since returns the time elapsed since the given time.
func (m *Mock) Since(t time.Time) time.Duration {
	return m.Now().Sub(t)
}

// Add progresses time by the given duration.
//
// It panics if the duration is negative.
func (m *Mock) Add(d time.Duration) {
	if d < 0 {
		panic("cannot add negative duration")
	}
	m.now = m.now.Add(d)
}
