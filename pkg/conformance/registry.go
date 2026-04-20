package conformance

import "fmt"

type Registry struct {
	Suites   map[string]Suite   `json:"suites"`
	Cases    map[string]Case    `json:"cases"`
	Features map[string]Feature `json:"features"`
}

func DefaultRegistry() Registry {
	features := map[string]Feature{
		"serve-introspection": {ID: "serve-introspection", Label: "Serve Introspection", Transport: []Transport{TransportServe}},
		"registration":        {ID: "registration", Label: "Registration"},
		"invocation":          {ID: "invocation", Label: "Invocation"},
		"steps":               {ID: "steps", Label: "Steps"},
		"retry":               {ID: "retry", Label: "Retry"},
		"cancel":              {ID: "cancel", Label: "Cancel"},
		"wait-for-event":      {ID: "wait-for-event", Label: "Wait For Event"},
		"malformed-input":     {ID: "malformed-input", Label: "Malformed Input Handling"},
		"auth-validation":     {ID: "auth-validation", Label: "Authentication Validation"},
		"secret-redaction":    {ID: "secret-redaction", Label: "Secret Redaction"},
		"env-nondisclosure":   {ID: "env-nondisclosure", Label: "Environment Variable Non-Disclosure"},
		"connect-readiness":   {ID: "connect-readiness", Label: "Connect Readiness", Transport: []Transport{TransportConnect}},
		"connect-reconnect":   {ID: "connect-reconnect", Label: "Connect Reconnect", Transport: []Transport{TransportConnect}},
		"connect-drain":       {ID: "connect-drain", Label: "Connect Drain Handling", Transport: []Transport{TransportConnect}},
		"connect-lease-renew": {ID: "connect-lease-renew", Label: "Connect Lease Extension", Transport: []Transport{TransportConnect}},
		"serve-registration":  {ID: "serve-registration", Label: "Serve Registration", Transport: []Transport{TransportServe}},
	}

	cases := map[string]Case{
		"serve-introspection": {
			ID: "serve-introspection", Label: "Serve Introspection", SuiteID: "transport-serve",
			Features: []string{"serve-introspection", "serve-registration"}, Transport: []Transport{TransportServe},
		},
		"basic-invoke": {
			ID: "basic-invoke", Label: "Basic Invoke", SuiteID: "core",
			Features: []string{"registration", "invocation"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"steps-serial": {
			ID: "steps-serial", Label: "Serial Steps", SuiteID: "core",
			Features: []string{"steps"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"retry-basic": {
			ID: "retry-basic", Label: "Retry", SuiteID: "core",
			Features: []string{"retry"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"cancel-basic": {
			ID: "cancel-basic", Label: "Cancel", SuiteID: "core",
			Features: []string{"cancel"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"wait-for-event-basic": {
			ID: "wait-for-event-basic", Label: "Wait For Event", SuiteID: "core",
			Features: []string{"wait-for-event"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"malformed-payload": {
			ID: "malformed-payload", Label: "Malformed Payload", SuiteID: "negative",
			Features: []string{"malformed-input"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"invalid-auth": {
			ID: "invalid-auth", Label: "Invalid Auth", SuiteID: "negative",
			Features: []string{"auth-validation"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"secret-redaction": {
			ID: "secret-redaction", Label: "Secret Redaction", SuiteID: "security",
			Features: []string{"secret-redaction", "env-nondisclosure"}, Transport: []Transport{TransportServe, TransportConnect},
		},
		"connect-ready": {
			ID: "connect-ready", Label: "Connect Ready", SuiteID: "transport-connect",
			Features: []string{"connect-readiness"}, Transport: []Transport{TransportConnect},
		},
		"connect-reconnect": {
			ID: "connect-reconnect", Label: "Connect Reconnect", SuiteID: "transport-connect",
			Features: []string{"connect-reconnect"}, Transport: []Transport{TransportConnect},
		},
		"connect-drain": {
			ID: "connect-drain", Label: "Connect Drain", SuiteID: "transport-connect",
			Features: []string{"connect-drain", "connect-lease-renew"}, Transport: []Transport{TransportConnect},
		},
	}

	suites := map[string]Suite{
		"core": {
			ID: "core", Label: "Core",
			Description: "Happy-path feature compatibility for generally implemented SDK capabilities.",
			CaseIDs:     []string{"basic-invoke", "steps-serial", "retry-basic", "cancel-basic", "wait-for-event-basic"},
		},
		"negative": {
			ID: "negative", Label: "Negative",
			Description: "Malformed and defensive-input coverage evaluated per feature.",
			CaseIDs:     []string{"malformed-payload", "invalid-auth"},
		},
		"security": {
			ID: "security", Label: "Security",
			Description: "Portable, externally observable security and secret-handling checks.",
			CaseIDs:     []string{"secret-redaction"},
		},
		"transport-serve": {
			ID: "transport-serve", Label: "Transport Serve",
			Description: "Serve-specific transport and registration cases.",
			CaseIDs:     []string{"serve-introspection"},
		},
		"transport-connect": {
			ID: "transport-connect", Label: "Transport Connect",
			Description: "Connect worker lifecycle and state-handling cases.",
			CaseIDs:     []string{"connect-ready", "connect-reconnect", "connect-drain"},
		},
	}

	registry := Registry{
		Suites:   suites,
		Cases:    cases,
		Features: features,
	}
	if err := registry.Validate(); err != nil {
		panic(fmt.Sprintf("invalid conformance registry: %v", err))
	}
	return registry
}

func (r Registry) Validate() error {
	for suiteID, suite := range r.Suites {
		if suite.ID == "" || suite.ID != suiteID {
			return fmt.Errorf("suite %q has mismatched id %q", suiteID, suite.ID)
		}
		for _, caseID := range suite.CaseIDs {
			c, ok := r.Cases[caseID]
			if !ok {
				return fmt.Errorf("suite %q references unknown case %q", suiteID, caseID)
			}
			if c.SuiteID != suiteID {
				return fmt.Errorf("case %q belongs to suite %q but was listed in %q", caseID, c.SuiteID, suiteID)
			}
		}
	}

	for caseID, testCase := range r.Cases {
		if testCase.ID == "" || testCase.ID != caseID {
			return fmt.Errorf("case %q has mismatched id %q", caseID, testCase.ID)
		}
		if _, ok := r.Suites[testCase.SuiteID]; !ok {
			return fmt.Errorf("case %q references unknown suite %q", caseID, testCase.SuiteID)
		}
		for _, featureID := range testCase.Features {
			if _, ok := r.Features[featureID]; !ok {
				return fmt.Errorf("case %q references unknown feature %q", caseID, featureID)
			}
		}
		for _, transport := range testCase.Transport {
			if !IsValidTransport(transport) {
				return fmt.Errorf("case %q uses invalid transport %q", caseID, transport)
			}
		}
	}

	for featureID, feature := range r.Features {
		if feature.ID == "" || feature.ID != featureID {
			return fmt.Errorf("feature %q has mismatched id %q", featureID, feature.ID)
		}
		for _, transport := range feature.Transport {
			if !IsValidTransport(transport) {
				return fmt.Errorf("feature %q uses invalid transport %q", featureID, transport)
			}
		}
	}

	return nil
}
