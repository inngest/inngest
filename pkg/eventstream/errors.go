package eventstream

import "fmt"

type ErrEventCount struct {
	Max int
}

func (e *ErrEventCount) Error() string {
	return fmt.Sprintf("Cannot have more than %d events in a batch", e.Max)
}
