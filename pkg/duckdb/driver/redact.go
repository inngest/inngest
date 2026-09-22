package driver

import (
	"errors"
	"strings"
	"sync"
)

const redactedPlaceholder = "<redacted>"

// secretRedactor scrubs registered secret values (quack auth tokens, a
// Postgres catalog URI) out of text before it reaches an error message or a
// log line. It exists because bootstrap statements carry those secrets inline
// (this transport has no parameter binding) and DuckDB echoes the offending
// statement back in some diagnostics — a Parser Error's "LINE 1: ..." excerpt,
// for example — so formatting only our own messages carefully isn't enough.
//
// A nil *secretRedactor is valid and redacts nothing.
type secretRedactor struct {
	mu      sync.RWMutex
	secrets []string
}

// add registers secrets to scrub. Empty values are ignored.
func (r *secretRedactor) add(secrets ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range secrets {
		if s != "" {
			r.secrets = append(r.secrets, s)
		}
	}
}

func (r *secretRedactor) redact(s string) string {
	if r == nil {
		return s
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, secret := range r.secrets {
		s = strings.ReplaceAll(s, secret, redactedPlaceholder)
	}
	return s
}

// redactErr returns err unchanged if its message contains no registered
// secret. Otherwise it returns an error with the scrubbed message that still
// matches the original chain under errors.Is (so errStatementFailed
// classification keeps working) but deliberately does not expose it via
// Unwrap, which would hand the unredacted message straight back.
func (r *secretRedactor) redactErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	scrubbed := r.redact(msg)
	if scrubbed == msg {
		return err
	}
	return &redactedError{msg: scrubbed, cause: err}
}

type redactedError struct {
	msg   string
	cause error
}

func (e *redactedError) Error() string { return e.msg }

func (e *redactedError) Is(target error) bool { return errors.Is(e.cause, target) }

// stmtSummary returns the leading keywords of a bootstrap statement (e.g.
// "INSTALL quack", "CALL quack_serve", "ATTACH IF NOT EXISTS",
// "CREATE OR REPLACE SECRET ducklake_quack_catalog") without any of its
// literals, so a failure can say which step failed without formatting a
// token or connection string into the error.
func stmtSummary(stmt string) string {
	var words []string
	for w := range strings.FieldsSeq(stmt) {
		if i := strings.IndexAny(w, "'\"(;="); i >= 0 {
			if i > 0 {
				words = append(words, w[:i])
			}
			break
		}
		words = append(words, w)
	}
	return strings.Join(words, " ")
}
