package rueidis

import "github.com/rueian/rueidis/internal/cmds"

// Completed represents a completed Redis command, should be created by the Build() of command builder.
type Completed = cmds.Completed

// Cacheable represents a completed Redis command which supports server-assisted client side caching,
// and it should be created by the Cache() of command builder.
type Cacheable = cmds.Cacheable

// Commands is an exported alias to []Completed.
// This allows users to store commands for later usage, for example:
//
//	c, release := client.Dedicate()
//	defer release()
//
//	cmds := make(rueidis.Commands, 0, 10)
//	for i := 0; i < 10; i++ {
//	    cmds = append(cmds, c.B().Set().Key(strconv.Itoa(i)).Value(strconv.Itoa(i)).Build())
//	}
//	for _, resp := range c.DoMulti(ctx, cmds...) {
//	    if err := resp.Error(); err != nil {
//	    panic(err)
//	}
//
// However, please know that once commands are processed by the Do() or DoMulti(), they are recycled and should not be reused.
type Commands []Completed
