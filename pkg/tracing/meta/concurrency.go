package meta

// CustomConcurrencyKey records the expression and evaluated value used by a
// custom step concurrency limit. A run can have more than one such limit.
type CustomConcurrencyKey struct {
	Scope      string `json:"scope"`
	Expression string `json:"expression"`
	Value      string `json:"value"`
}
