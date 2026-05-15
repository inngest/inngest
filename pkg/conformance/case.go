package conformance

type Case struct {
	ID          string      `json:"id" koanf:"id"`
	Label       string      `json:"label" koanf:"label"`
	Description string      `json:"description,omitempty" koanf:"description"`
	SuiteID     string      `json:"suite_id" koanf:"suite-id"`
	Features    []string    `json:"features" koanf:"features"`
	Transport   []Transport `json:"transport,omitempty" koanf:"transport"`
}

func (c Case) SupportsTransport(transport Transport) bool {
	if transport == "" {
		return true
	}
	if len(c.Transport) == 0 {
		return true
	}
	for _, candidate := range c.Transport {
		if candidate == transport {
			return true
		}
	}
	return false
}
