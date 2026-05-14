package queue

import (
	"fmt"

	"github.com/google/uuid"
)

func (s Scope) Validate() error {
	if s.IsSystem {
		return nil
	}
	if s.AccountID == uuid.Nil {
		return fmt.Errorf("missing account ID")
	}
	if s.EnvID == uuid.Nil {
		return fmt.Errorf("missing env ID")
	}
	if s.FunctionID == uuid.Nil {
		return fmt.Errorf("missing function ID")
	}
	return nil
}
