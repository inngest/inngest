package output

import (
	"fmt"
	"sort"
	"strings"

	conf "github.com/inngest/inngest/pkg/conformance"
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
