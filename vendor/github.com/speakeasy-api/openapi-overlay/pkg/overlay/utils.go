package overlay

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

func NewTargetSelector(path, method string) string {
	return fmt.Sprintf(`$["paths"]["%s"]["%s"]`, path, method)
}

func NewUpdateAction(path, method string, update yaml.Node) Action {
	target := NewTargetSelector(path, method)

	return Action{
		Target: target,
		Update: update,
	}
}
