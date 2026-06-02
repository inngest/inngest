package conformance

type Suite struct {
	ID          string   `json:"id" koanf:"id"`
	Label       string   `json:"label" koanf:"label"`
	Description string   `json:"description,omitempty" koanf:"description"`
	CaseIDs     []string `json:"case_ids" koanf:"case-ids"`
}
