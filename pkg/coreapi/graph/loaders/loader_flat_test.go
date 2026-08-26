package loader

import (
	"testing"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/stretchr/testify/require"
)

type flatSpanManagerStub struct{ cqrs.Manager }

func (flatSpanManagerStub) FlatSpans() bool { return true }

func TestNewLoadersSelectsFlatConverterForFlatSpanSource(t *testing.T) {
	loaders := NewLoaders(LoaderParams{DB: flatSpanManagerStub{}})
	require.NotNil(t, loaders.RunTraceLoader)
}
