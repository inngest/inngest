package store

// InvalidateOption represents a cache invalidation function.
type InvalidateOption func(o *InvalidateOptions)

type InvalidateOptions struct {
	Tags []string
}

func (o *InvalidateOptions) isEmpty() bool {
	return len(o.Tags) == 0
}

func ApplyInvalidateOptionsWithDefault(defaultOptions *InvalidateOptions, opts ...InvalidateOption) *InvalidateOptions {
	returnedOptions := ApplyInvalidateOptions(opts...)

	if returnedOptions == new(InvalidateOptions) {
		returnedOptions = defaultOptions
	}

	return returnedOptions
}

func ApplyInvalidateOptions(opts ...InvalidateOption) *InvalidateOptions {
	o := &InvalidateOptions{}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// WithInvalidateTags allows setting the invalidate tags.
func WithInvalidateTags(tags []string) InvalidateOption {
	return func(o *InvalidateOptions) {
		o.Tags = tags
	}
}
