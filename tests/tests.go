package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Invoke via: gp run ./*.go -test.v
func main() {
	ctx := context.Background()
	do(ctx)
}

func do(ctx context.Context) {
	testing.Init()

	cfgs, err := parseConfigs(ctx)
	if err != nil {
		panic(err.Error())
	}

	fns, err := parseFns(ctx)
	if err != nil {
		panic(err.Error())
	}

	tests := []testing.InternalTest{}

	for _, cfg := range cfgs {
		copiedCfg := *cfg
		// Run each test.
		for _, item := range fns {
			copiedFn := *item
			fn := &copiedFn

			// Create a new test for each cfg/fn pair
			tests = append(tests, testing.InternalTest{
				Name: fmt.Sprintf("%s/%s", filepath.Base(copiedCfg.dir), filepath.Base(fn.dir)),
				F: func(t *testing.T) {
					fmt.Println("")
					cmdCtx, done := context.WithCancel(ctx)
					defer done()

					// TODO: Instead of exec-ing the service, run the services locally
					// then profile.
					err := copiedCfg.Up(cmdCtx)
					require.NoError(t, err)

					err = fn.Test(ctx, copiedCfg)
					fmt.Println("")
					require.NoError(t, err, "Output:\n%s", copiedCfg.out.String())
				},
			})

		}
	}

	t := &TestDeps{}
	m := testing.MainStart(t, tests, nil, nil, nil)
	os.Exit(m.Run())
}

type cmdError struct {
	err error
	out []byte
}

func (c cmdError) Error() string {
	s := strings.Builder{}
	_, _ = s.WriteString(fmt.Sprintf("Command failed with error: %s", c.err.Error()) + "\n")
	_, _ = s.WriteString(fmt.Sprintf("Output: \n%s", string(c.out)+"\n"))
	return s.String()
}
