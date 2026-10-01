package main

import (
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
)

func TestIncludeValues(t *testing.T) {
	method := apiv2.File_api_v2_service_proto.Services().ByName("V2").Methods().ByName("ListRuns")
	values := includeValues(method)
	require.Equal(t, []string{"output", "deferredFrom"}, values)
	require.Equal(t, "ListRunsInclude", includeTypeName(method))
}

func TestNewSelectorSpec(t *testing.T) {
	spec, err := newSelectorSpec("ListRuns", "ListRunsInclude", []string{"output", "deferredFrom", "summaryDetails"})
	require.NoError(t, err)
	require.Equal(t, "ListRunsInclude", spec.typeName)
	require.Equal(t, []string{"output", "deferredFrom", "summaryDetails"}, spec.values)
	require.Equal(t, map[string]string{
		"output":          "output",
		"deferredFrom":    "deferredFrom",
		"deferred_from":   "deferredFrom",
		"summaryDetails":  "summaryDetails",
		"summary_details": "summaryDetails",
	}, spec.aliases)
}

func TestNewSelectorSpecRejectsSnakeCase(t *testing.T) {
	_, err := newSelectorSpec("ListRuns", "ListRunsInclude", []string{"deferred_from"})
	require.ErrorContains(t, err, `include value "deferred_from" for ListRuns must be lowerCamelCase`)
}

func TestEmitSelectorsGeneratesIndependentTypes(t *testing.T) {
	runs, err := newSelectorSpec("ListRuns", "ListRunsInclude", []string{"output"})
	require.NoError(t, err)
	trace, err := newSelectorSpec("GetTrace", "GetTraceInclude", []string{"output"})
	require.NoError(t, err)

	plugin := &protogen.Plugin{}
	generated := plugin.NewGeneratedFile("include_selectors.go", generatedPackage)
	generated.P("package apiv2")
	require.NoError(t, emitSelectors(generated, []selectorSpec{runs, trace}))
	content, err := generated.Content()
	require.NoError(t, err)
	require.Contains(t, string(content), "type ListRunsInclude uint8")
	require.Contains(t, string(content), "type GetTraceInclude uint8")
	require.Contains(t, string(content), "map[string]ListRunsInclude")
	require.Contains(t, string(content), "map[string]GetTraceInclude")
}

func TestGoName(t *testing.T) {
	require.Equal(t, "DeferredFrom", goName("deferredFrom"))
	require.Equal(t, "TotalCount", goName("totalCount"))
}
