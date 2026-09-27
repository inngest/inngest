package main

import (
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestNewSelectorSpec(t *testing.T) {
	method := apiv2.File_api_v2_service_proto.Services().ByName("V2").Methods().ByName("ListRuns")
	spec, err := newSelectorSpec("ListRuns", method, []string{"output", "deferredFrom"})
	require.NoError(t, err)
	require.Equal(t, []string{"output", "deferredFrom"}, spec.values)
	require.Equal(t, map[string]string{
		"output":        "output",
		"deferredFrom":  "deferredFrom",
		"deferred_from": "deferredFrom",
	}, spec.aliases)
}

func TestNewSelectorSpecRejectsUnknownResponseField(t *testing.T) {
	method := apiv2.File_api_v2_service_proto.Services().ByName("V2").Methods().ByName("ListRuns")
	_, err := newSelectorSpec("ListRuns", method, []string{"notAResponseField"})
	require.ErrorContains(t, err, `include value "notAResponseField" does not name a response field`)
}

func TestGoName(t *testing.T) {
	require.Equal(t, "DeferredFrom", goName("deferredFrom"))
	require.Equal(t, "TotalCount", goName("total_count"))
}
