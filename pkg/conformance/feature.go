package conformance

type Feature struct {
	ID          string      `json:"id" koanf:"id"`
	Label       string      `json:"label" koanf:"label"`
	Description string      `json:"description,omitempty" koanf:"description"`
	Transport   []Transport `json:"transport,omitempty" koanf:"transport"`
}
