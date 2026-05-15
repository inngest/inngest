package output

import (
	"fmt"
	"sort"
	"strings"

	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/inngest/inngest/pkg/conformance/runner/serve"
)

func TextConformanceCatalog(registry conf.Registry) error {
	w := NewTextWriter()

	transports := make([]string, 0, len(conf.ValidTransports()))
	for _, transport := range conf.ValidTransports() {
		transports = append(transports, string(transport))
	}

	root := OrderedData(
		"Transports", strings.Join(transports, ", "),
	)

	suiteIDs := make([]string, 0, len(registry.Suites))
	for suiteID := range registry.Suites {
		suiteIDs = append(suiteIDs, suiteID)
	}
	sort.Strings(suiteIDs)

	suites := NewOrderedMap()
	for _, suiteID := range suiteIDs {
		suite := registry.Suites[suiteID]
		caseIDs := make([]string, 0, len(suite.CaseIDs))
		caseIDs = append(caseIDs, suite.CaseIDs...)
		sort.Strings(caseIDs)
		suites.Set(suite.ID, OrderedData(
			"Label", suite.Label,
			"Description", suite.Description,
			"Cases", strings.Join(caseIDs, ", "),
		))
	}
	root.Set("Suites", suites)

	featureIDs := make([]string, 0, len(registry.Features))
	for featureID := range registry.Features {
		featureIDs = append(featureIDs, featureID)
	}
	sort.Strings(featureIDs)

	features := NewOrderedMap()
	for _, featureID := range featureIDs {
		feature := registry.Features[featureID]
		transports := make([]string, 0, len(feature.Transport))
		for _, transport := range feature.Transport {
			transports = append(transports, string(transport))
		}

		data := OrderedData("Label", feature.Label)
		if feature.Description != "" {
			data.Set("Description", feature.Description)
		}
		if len(transports) > 0 {
			sort.Strings(transports)
			data.Set("Transports", strings.Join(transports, ", "))
		}
		features.Set(feature.ID, data)
	}
	root.Set("Features", features)

	if err := w.WriteOrdered(root, WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}

func TextConformanceRunPlan(plan conf.RunPlan) error {
	w := NewTextWriter()

	transport := "all"
	if plan.Transport != "" {
		transport = string(plan.Transport)
	}

	root := OrderedData(
		"Transport", transport,
		"Suite Count", len(plan.Suites),
		"Case Count", len(plan.Cases),
		"Feature Count", len(plan.Features),
	)

	suites := NewOrderedMap()
	for _, suite := range plan.Suites {
		suites.Set(suite.ID, OrderedData(
			"Label", suite.Label,
			"Description", suite.Description,
		))
	}
	root.Set("Resolved Suites", suites)

	cases := NewOrderedMap()
	for _, testCase := range plan.Cases {
		transports := make([]string, 0, len(testCase.Transport))
		for _, transport := range testCase.Transport {
			transports = append(transports, string(transport))
		}
		sort.Strings(transports)

		data := OrderedData(
			"Label", testCase.Label,
			"Suite", testCase.SuiteID,
			"Features", strings.Join(testCase.Features, ", "),
		)
		if len(transports) > 0 {
			data.Set("Transports", strings.Join(transports, ", "))
		}
		cases.Set(testCase.ID, data)
	}
	root.Set("Resolved Cases", cases)

	features := NewOrderedMap()
	for _, feature := range plan.Features {
		features.Set(feature.ID, feature.Label)
	}
	root.Set("Resolved Features", features)

	root.Set("Compatibility Classes", strings.Join([]string{
		string(conf.CompatibilityFull),
		string(conf.CompatibilityPartial),
		string(conf.CompatibilityIncompatible),
		string(conf.CompatibilityUnknown),
	}, ", "))
	root.Set("Note", "Phase 1 validates configuration and selection only. Execution is not implemented yet.")

	if err := w.WriteOrdered(root, WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}

func TextConformanceStub(command, message string) error {
	w := NewTextWriter()
	if err := w.WriteOrdered(OrderedData(
		"Command", command,
		"Status", "stub",
		"Message", message,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}

func TextConformanceConfigErrorHint(configPath string) error {
	msg := "No config file provided."
	if configPath != "" {
		msg = fmt.Sprintf("Config loaded from %s.", configPath)
	}
	return TextConformanceStub("conformance", msg)
}

func TextConformanceDoctor(checks []serve.Check) error {
	w := NewTextWriter()

	root := NewOrderedMap()
	for _, check := range checks {
		status := "failed"
		if check.Passed {
			status = "passed"
		}

		root.Set(check.Name, OrderedData(
			"Status", status,
			"Message", check.Message,
		))
	}

	if err := w.WriteOrdered(OrderedData("Checks", root), WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}

func TextConformanceReport(report conf.Report) error {
	w := NewTextWriter()

	root := OrderedData(
		"Schema Version", report.SchemaVersion,
		"Transport", string(report.Transport),
		"Compatibility", string(report.Compatibility),
		"Case Count", len(report.Cases),
		"Feature Count", len(report.Features),
	)

	caseResults := NewOrderedMap()
	caseIDs := make([]string, 0, len(report.Cases))
	caseByID := map[string]conf.CaseResult{}
	for _, result := range report.Cases {
		caseIDs = append(caseIDs, result.CaseID)
		caseByID[result.CaseID] = result
	}
	sort.Strings(caseIDs)
	for _, caseID := range caseIDs {
		result := caseByID[caseID]
		data := OrderedData(
			"Suite", result.SuiteID,
			"Status", string(result.Status),
		)
		if result.ReasonCode != "" {
			data.Set("Reason Code", string(result.ReasonCode))
		}
		if result.Reason != "" {
			data.Set("Reason", result.Reason)
		}
		caseResults.Set(caseID, data)
	}
	root.Set("Cases", caseResults)

	featureResults := NewOrderedMap()
	featureIDs := make([]string, 0, len(report.Features))
	featureByID := map[string]conf.FeatureResult{}
	for _, result := range report.Features {
		featureIDs = append(featureIDs, result.FeatureID)
		featureByID[result.FeatureID] = result
	}
	sort.Strings(featureIDs)
	for _, featureID := range featureIDs {
		result := featureByID[featureID]
		data := OrderedData(
			"Compatibility", string(result.Compatibility),
			"Backing Cases", strings.Join(result.BackingCaseIDs, ", "),
		)
		if result.ReasonCode != "" {
			data.Set("Reason Code", string(result.ReasonCode))
		}
		if result.Reason != "" {
			data.Set("Reason", result.Reason)
		}
		featureResults.Set(featureID, data)
	}
	root.Set("Features", featureResults)

	if err := w.WriteOrdered(root, WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}
