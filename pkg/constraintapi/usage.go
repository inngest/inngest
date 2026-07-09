package constraintapi

type scriptConstraintUsage struct {
	Index      int                      `json:"i,omitempty"`
	Used       int                      `json:"u"`
	Limit      int                      `json:"l"`
	Constraint SerializedConstraintItem `json:"c,omitempty"`
}

func constraintItemsFromSerialized(values []SerializedConstraintItem) []ConstraintItem {
	if len(values) == 0 {
		return nil
	}

	constraints := make([]ConstraintItem, len(values))
	for i, v := range values {
		constraints[i] = v.ToConstraintItem()
	}
	return constraints
}

func constraintUsageFromScript(values []scriptConstraintUsage, constraints []ConstraintItem) []ConstraintUsage {
	if len(values) == 0 {
		return nil
	}

	usage := make([]ConstraintUsage, 0, len(values))
	for i, v := range values {
		constraint := v.Constraint.ToConstraintItem()
		if constraint.Kind == "" {
			index := v.Index
			if index == 0 {
				index = i + 1
			}
			if index < 1 || index > len(constraints) {
				continue
			}
			constraint = constraints[index-1]
		}
		if constraint.Kind == "" {
			continue
		}

		usage = append(usage, ConstraintUsage{
			Constraint: constraint,
			Used:       v.Used,
			Limit:      v.Limit,
		})
	}

	return usage
}
