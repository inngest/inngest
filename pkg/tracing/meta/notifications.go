package meta

import (
	"sync"
)

var (
	spanNotifications = make(map[string]chan error)
	notificationMu    sync.RWMutex
)

// RegisterSpanNotification creates and registers a notification channel for the
// given span ID. The caller should wait on the returned channel to be notified
// when a span has been exported.
func RegisterSpanNotification(spanID string) <-chan error {
	notificationMu.Lock()
	defer notificationMu.Unlock()

	ch := make(chan error, 1)
	spanNotifications[spanID] = ch
	return ch
}

// NotifySpanExported notifies any waiting goroutine that a span has been
// exported. This should be called after the span has been successfully exported
// (or failed to export).
func NotifySpanExported(spanID string, err error) {
	notificationMu.Lock()
	ch, exists := spanNotifications[spanID]
	if exists {
		delete(spanNotifications, spanID)
	}
	notificationMu.Unlock()

	if exists {
		ch <- err
	}
}

// CleanupSpanNotification removes a span notification without sending a result.
// This is useful for cleanup in error scenarios.
func CleanupSpanNotification(spanID string) {
	notificationMu.Lock()
	delete(spanNotifications, spanID)
	notificationMu.Unlock()
}
