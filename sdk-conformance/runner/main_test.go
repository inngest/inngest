package main

import "testing"

func TestCheckAssertionDistinguishesAbsentFromNull(t *testing.T) {
	document := map[string]any{"present_null": nil}

	if err := checkAssertion(document, assertion{Path: "$.present_null", Operator: "present"}); err != nil {
		t.Fatalf("present null value should be present: %v", err)
	}
	if err := checkAssertion(document, assertion{Path: "$.missing", Operator: "absent"}); err != nil {
		t.Fatalf("missing value should be absent: %v", err)
	}
	if err := checkAssertion(document, assertion{Path: "$.present_null", Operator: "type", Type: "null"}); err != nil {
		t.Fatalf("present null value should have null type: %v", err)
	}
}

func TestCheckAssertionMatchesNestedValue(t *testing.T) {
	document := map[string]any{"sdk": map[string]any{"language": "go"}}
	if err := checkAssertion(document, assertion{Path: "$.sdk.language", Operator: "equals", Value: "go"}); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedRequiresProfileAndEveryCapability(t *testing.T) {
	tc := testCase{
		Profiles:     []string{"serve.execution.v2"},
		Capabilities: []string{"step.run", "step.sleep"},
	}
	target := targetManifest{
		Profiles:     []string{"serve.execution.v2"},
		Capabilities: []string{"step.run"},
	}

	missing := unsupported(tc, target)
	if len(missing) != 1 || missing[0] != "step.sleep" {
		t.Fatalf("missing = %v, want [step.sleep]", missing)
	}
}
