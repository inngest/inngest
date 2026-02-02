package redis_state

import "github.com/inngest/inngest/pkg/util"

var rnd *util.FrandRNG

func init() {
	// For weighted shuffles generate a new rand.
	rnd = util.NewFrandRNG()
}
