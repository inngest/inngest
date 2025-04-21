package dnscache

import "log/slog"

type Option struct {
	apply func(r *Resolver)
}

func WithLogger(logger *slog.Logger) Option {
	return Option{apply: func(r *Resolver) {
		r.logger = logger
	}}
}
