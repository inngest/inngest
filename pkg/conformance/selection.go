package conformance

import (
	"fmt"
	"slices"
	"sort"
)

type Selection struct {
	Transport Transport
	Suites    []string
	Cases     []string
	Features  []string
}

type RunPlan struct {
	Transport Transport `json:"transport,omitempty"`
	Suites    []Suite   `json:"suites"`
	Cases     []Case    `json:"cases"`
	Features  []Feature `json:"features"`
}

func (c Config) Selection() Selection {
	return Selection{
		Transport: c.Transport,
		Suites:    slices.Clone(c.Suites),
		Cases:     slices.Clone(c.Cases),
		Features:  slices.Clone(c.Features),
	}
}

func (s Selection) Resolve(registry Registry) (RunPlan, error) {
	if s.Transport != "" && !IsValidTransport(s.Transport) {
		return RunPlan{}, fmt.Errorf("unknown transport %q", s.Transport)
	}

	for _, suiteID := range s.Suites {
		if _, ok := registry.Suites[suiteID]; !ok {
			return RunPlan{}, fmt.Errorf("unknown suite %q", suiteID)
		}
	}
	for _, caseID := range s.Cases {
		if _, ok := registry.Cases[caseID]; !ok {
			return RunPlan{}, fmt.Errorf("unknown case %q", caseID)
		}
	}
	for _, featureID := range s.Features {
		if _, ok := registry.Features[featureID]; !ok {
			return RunPlan{}, fmt.Errorf("unknown feature %q", featureID)
		}
	}

	selectedCaseIDs := map[string]struct{}{}
	if len(s.Suites) == 0 && len(s.Cases) == 0 {
		for caseID := range registry.Cases {
			selectedCaseIDs[caseID] = struct{}{}
		}
	}

	for _, suiteID := range s.Suites {
		for _, caseID := range registry.Suites[suiteID].CaseIDs {
			selectedCaseIDs[caseID] = struct{}{}
		}
	}
	for _, caseID := range s.Cases {
		selectedCaseIDs[caseID] = struct{}{}
	}

	filteredCaseIDs := make([]string, 0, len(selectedCaseIDs))
	for caseID := range selectedCaseIDs {
		testCase := registry.Cases[caseID]
		if !testCase.SupportsTransport(s.Transport) {
			continue
		}
		if len(s.Features) > 0 && !caseHasAnyFeature(testCase, s.Features) {
			continue
		}
		filteredCaseIDs = append(filteredCaseIDs, caseID)
	}
	sort.Strings(filteredCaseIDs)

	if len(filteredCaseIDs) == 0 {
		return RunPlan{}, fmt.Errorf("selection did not match any conformance cases")
	}

	suiteIDs := make(map[string]struct{})
	featureIDs := make(map[string]struct{})
	planCases := make([]Case, 0, len(filteredCaseIDs))
	for _, caseID := range filteredCaseIDs {
		testCase := registry.Cases[caseID]
		planCases = append(planCases, testCase)
		suiteIDs[testCase.SuiteID] = struct{}{}
		for _, featureID := range testCase.Features {
			featureIDs[featureID] = struct{}{}
		}
	}

	planSuites := make([]Suite, 0, len(suiteIDs))
	for suiteID := range suiteIDs {
		planSuites = append(planSuites, registry.Suites[suiteID])
	}
	sort.Slice(planSuites, func(i, j int) bool { return planSuites[i].ID < planSuites[j].ID })

	planFeatures := make([]Feature, 0, len(featureIDs))
	for featureID := range featureIDs {
		planFeatures = append(planFeatures, registry.Features[featureID])
	}
	sort.Slice(planFeatures, func(i, j int) bool { return planFeatures[i].ID < planFeatures[j].ID })

	return RunPlan{
		Transport: s.Transport,
		Suites:    planSuites,
		Cases:     planCases,
		Features:  planFeatures,
	}, nil
}

func caseHasAnyFeature(testCase Case, features []string) bool {
	for _, featureID := range testCase.Features {
		if slices.Contains(features, featureID) {
			return true
		}
	}
	return false
}
