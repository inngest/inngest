package store

import (
	"fmt"
	"time"

	"golang.org/x/exp/slices"
)

type OptionsMatcher struct {
	Cost       int64
	Expiration time.Duration
	Tags       []string
}

func (m OptionsMatcher) Matches(x interface{}) bool {
	switch values := x.(type) {
	case []Option:
		opts := &Options{}
		for _, value := range values {
			value(opts)
		}

		return opts.Cost == m.Cost &&
			opts.Expiration == m.Expiration &&
			slices.Equal(opts.Tags, m.Tags)
	}

	return false
}

func (m OptionsMatcher) String() string {
	return fmt.Sprintf(
		"options should match (cost: %v expiration: %v tags: %v)",
		m.Cost,
		m.Expiration,
		m.Tags,
	)
}

type InvalidateOptionsMatcher struct {
	Tags []string
}

func (m InvalidateOptionsMatcher) Matches(x interface{}) bool {
	switch values := x.(type) {
	case []InvalidateOption:
		opts := &InvalidateOptions{}
		for _, value := range values {
			value(opts)
		}

		return slices.Equal(opts.Tags, m.Tags)
	}

	return false
}

func (m InvalidateOptionsMatcher) String() string {
	return fmt.Sprintf(
		"invalidate options should match (tags: %v)",
		m.Tags,
	)
}
