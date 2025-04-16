package inngest

import (
	"bytes"
	"github.com/google/uuid"
)

func GetFailureHandlerSlug(functionSlug string) string {
	return functionSlug + "-failure"
}

func DeterministicUUIDV7(str string) (uuid.UUID, error) {
	randomness := bytes.NewBufferString(str)

	return uuid.NewV7FromReader(randomness)
}
