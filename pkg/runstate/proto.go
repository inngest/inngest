package runstate

import (
	statev1 "github.com/inngest/inngest/proto/gen/state/v1"
)

func StateFromProto(s *statev1.State) State {
	return State{}
}
