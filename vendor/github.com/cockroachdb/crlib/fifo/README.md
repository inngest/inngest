## Go facilities for FIFO queueing

This library contains several optimized facilities related to FIFO queueing and
rate limiting.

 - [Queue](https://github.com/cockroachdb/crlib/blob/main/fifo/queue.go) implements an
   allocation efficient FIFO queue.

 - [Semaphore](https://github.com/cockroachdb/crlib/blob/main/fifo/semaphore.go)
   implements a weighted, dynamically reconfigurable semaphore which respects
   context cancellation.

TODO(radu): add rate limiter.
