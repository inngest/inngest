package conformance

// BuildFeatureResults rolls up case outcomes into per-feature compatibility.
//
// The compatibility model intentionally stays coarse in Phase 2:
// - any failed backing case makes the feature incompatible
// - any mix of passed and unevaluable/not-implemented becomes partial
// - all passed becomes full
// - no meaningful evaluation remains unknown
func BuildFeatureResults(plan RunPlan, caseResults []CaseResult) []FeatureResult {
	caseByID := make(map[string]CaseResult, len(caseResults))
	for _, result := range caseResults {
		caseByID[result.CaseID] = result
	}

	features := make([]FeatureResult, 0, len(plan.Features))
	for _, feature := range plan.Features {
		backing := backingCasesForFeature(plan.Cases, feature.ID)
		featureResult := FeatureResult{
			FeatureID:      feature.ID,
			BackingCaseIDs: backing,
			Compatibility:  CompatibilityUnknown,
		}

		var (
			passed bool
			failed bool
			other  bool
		)

		for _, caseID := range backing {
			result, ok := caseByID[caseID]
			if !ok {
				continue
			}

			switch result.Status {
			case CaseStatusPassed:
				passed = true
			case CaseStatusFailed:
				failed = true
				if featureResult.ReasonCode == "" {
					featureResult.ReasonCode = result.ReasonCode
					featureResult.Reason = result.Reason
				}
			default:
				other = true
				if featureResult.ReasonCode == "" {
					featureResult.ReasonCode = result.ReasonCode
					featureResult.Reason = result.Reason
				}
			}
		}

		switch {
		case failed:
			featureResult.Compatibility = CompatibilityIncompatible
		case passed && !other:
			featureResult.Compatibility = CompatibilityFull
		case passed && other:
			featureResult.Compatibility = CompatibilityPartial
		default:
			featureResult.Compatibility = CompatibilityUnknown
		}

		features = append(features, featureResult)
	}

	return features
}

// OverallCompatibility computes the top-level SDK compatibility from feature
// outcomes, keeping feature-level evaluation as the source of truth.
func OverallCompatibility(features []FeatureResult) Compatibility {
	if len(features) == 0 {
		return CompatibilityUnknown
	}

	var (
		hasFull    bool
		hasPartial bool
		hasUnknown bool
	)

	for _, feature := range features {
		switch feature.Compatibility {
		case CompatibilityIncompatible:
			return CompatibilityIncompatible
		case CompatibilityFull:
			hasFull = true
		case CompatibilityPartial:
			hasPartial = true
		case CompatibilityUnknown:
			hasUnknown = true
		}
	}

	switch {
	case hasPartial:
		return CompatibilityPartial
	case hasFull && hasUnknown:
		return CompatibilityPartial
	case hasFull:
		return CompatibilityFull
	default:
		return CompatibilityUnknown
	}
}

func backingCasesForFeature(cases []Case, featureID string) []string {
	backing := make([]string, 0, len(cases))
	for _, testCase := range cases {
		for _, caseFeatureID := range testCase.Features {
			if caseFeatureID == featureID {
				backing = append(backing, testCase.ID)
				break
			}
		}
	}
	return backing
}
