package golang

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(t *testing.T) {
	goleak.VerifyTestMain(t)
}
