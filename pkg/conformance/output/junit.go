package output

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/inngest/inngest/pkg/conformance"
)

// RenderJUnit converts a conformance Report into JUnit XML suitable for CI systems
// such as GitHub Actions, GitLab CI, and Jenkins.
//
// Design goals for v1:
//   - One <testsuite> per conformance suite_id (e.g. "core", "negative").
//   - One <testcase> per conformance case_id.
//   - Map conformance statuses to JUnit elements in a CI-friendly way:
//     passed          -> plain <testcase>
//     failed          -> <failure>   (should fail the pipeline)
//     not_evaluable   -> <error>     (environment/setup problem)
//     not_implemented -> <skipped>   (SDK gap, not a regression)
//     skipped         -> <skipped>
//
// Feature-level rollups from report.Features are intentionally omitted in v1.
// CI tools operate on testcases; features remain in the JSON artifact.
func RenderJUnit(report conformance.Report) ([]byte, error) {
	root := junitTestsuites{
		Name: "inngest-conformance",
	}

	// Group cases by suite so CI output mirrors the conformance suite layout.
	suiteCases := groupCasesBySuite(report.Cases)

	// Stable ordering makes output deterministic across runs and platforms.
	suiteIDs := sortedKeys(suiteCases)

	for _, suiteID := range suiteIDs {
		cases := suiteCases[suiteID]
		suite := buildTestsuite(suiteID, report.Transport, cases)
		root.addSuite(suite)
	}

	var buf bytes.Buffer

	// XML declaration is required by many CI parsers even though encoding/xml
	// does not emit it automatically.
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("encode junit xml: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("flush junit xml: %w", err)
	}

	return buf.Bytes(), nil
}

// --- XML model (kept local to avoid coupling to vendor junit helpers) ---

type junitTestsuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Skipped  int              `xml:"skipped,attr"`
	Suites   []junitTestsuite `xml:"testsuite"`
}

func (t *junitTestsuites) addSuite(s junitTestsuite) {
	t.Suites = append(t.Suites, s)
	t.Tests += s.Tests
	t.Failures += s.Failures
	t.Errors += s.Errors
	t.Skipped += s.Skipped
}

type junitTestsuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Skipped  int             `xml:"skipped,attr"`
	Cases    []junitTestcase `xml:"testcase"`
}

type junitTestcase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Classname string        `xml:"classname,attr"`
	Name      string        `xml:"name,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Error     *junitError   `xml:"error,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Body    string `xml:",chardata"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// buildTestsuite materializes one JUnit testsuite from all cases in a conformance suite.
func buildTestsuite(suiteID string, transport conformance.Transport, cases []conformance.CaseResult) junitTestsuite {
	suite := junitTestsuite{
		Name: suiteID,
	}

	// Sort cases by ID for stable XML ordering (matches pretty printer behavior).
	sort.Slice(cases, func(i, j int) bool {
		return cases[i].CaseID < cases[j].CaseID
	})

	for _, result := range cases {
		tc := junitTestcase{
			// Classname gives CI UIs a hierarchical name: inngest.conformance.<suite>.<transport>
			Classname: junitClassname(suiteID, transport),
			Name:      result.CaseID,
		}

		switch result.Status {
		case conformance.CaseStatusPassed:
			// No child element: JUnit interprets this as a passing test.

		case conformance.CaseStatusFailed:
			suite.Failures++
			tc.Failure = &junitFailure{
				Message: junitMessage(result),
				Type:    string(result.ReasonCode),
				Body:    result.Reason,
			}

		case conformance.CaseStatusNotEvaluable:
			// Treat transport/setup failures as <error> so CI alerts infra problems.
			suite.Errors++
			tc.Error = &junitError{
				Message: junitMessage(result),
				Type:    string(result.ReasonCode),
				Body:    result.Reason,
			}

		case conformance.CaseStatusNotImplemented, conformance.CaseStatusSkipped:
			// Missing SDK support is not a regression; mark as skipped.
			suite.Skipped++
			tc.Skipped = &junitSkipped{
				Message: junitMessage(result),
			}

		default:
			// Unknown future status: fail safe as error so CI does not silently pass.
			suite.Errors++
			tc.Error = &junitError{
				Message: fmt.Sprintf("unknown case status %q", result.Status),
				Type:    "unknown_status",
			}
		}

		suite.Cases = append(suite.Cases, tc)
		suite.Tests++
	}

	return suite
}

// junitClassname builds a stable, dot-separated classname for CI dashboards.
func junitClassname(suiteID string, transport conformance.Transport) string {
	if transport == "" {
		return "inngest.conformance." + suiteID
	}
	return fmt.Sprintf("inngest.conformance.%s.%s", suiteID, transport)
}

// junitMessage prefers the human reason, then falls back to the machine reason code.
func junitMessage(result conformance.CaseResult) string {
	if result.Reason != "" {
		return result.Reason
	}
	if result.ReasonCode != "" {
		return string(result.ReasonCode)
	}
	return string(result.Status)
}

// groupCasesBySuite indexes case results by their suite_id field.
func groupCasesBySuite(cases []conformance.CaseResult) map[string][]conformance.CaseResult {
	out := make(map[string][]conformance.CaseResult, len(cases))
	for _, result := range cases {
		suiteID := result.SuiteID
		if suiteID == "" {
			// Fallback bucket for malformed reports; should not happen in practice.
			suiteID = "unknown"
		}
		out[suiteID] = append(out[suiteID], result)
	}
	return out
}

// sortedKeys returns map keys in lexicographic order.
func sortedKeys[V any](m map[string][]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
