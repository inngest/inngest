package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	inmemorydatastore "github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// Invoke via: go run ./*.go -test.v
func main() {
	ctx := context.Background()

	// Ensure that all images within `tests/fns` have been built.
	if err := build(ctx); err != nil {
		panic(fmt.Sprintf("error building images: %s", err))
	}

	fmt.Println("")
	fmt.Println("")
	do(ctx)
}

func build(ctx context.Context) error {
	fmt.Println("Building images...")
	// Create a new filesystem loader.
	el, err := inmemorydatastore.NewFSLoader(ctx, ".")
	if err != nil {
		return err
	}

	funcs, err := el.Functions(ctx)
	if err != nil {
		return err
	}

	opts := []dockerdriver.BuildOpts{}
	for _, fn := range funcs {
		steps, err := dockerdriver.FnBuildOpts(ctx, fn)
		if err != nil {
			return err
		}
		opts = append(opts, steps...)
	}

	if len(opts) == 0 {
		return nil
	}

	eg := errgroup.Group{}
	for _, opt := range opts {
		copied := opt
		eg.Go(func() error {
			fmt.Printf("Building %s\n", copied.Path)
			b, err := dockerdriver.NewBuilder(ctx, copied)
			if err != nil {
				return err
			}
			if err := b.Start(); err != nil {
				return err
			}
			return b.Wait()
		})
	}

	return eg.Wait()
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
			name := fmt.Sprintf("%s-%s", filepath.Base(copiedCfg.dir), filepath.Base(fn.dir))

			// Create a new test for each cfg/fn pair
			tests = append(tests, testing.InternalTest{
				Name: name,
				F: func(t *testing.T) {
					fmt.Println("")
					cmdCtx, done := context.WithCancel(ctx)
					defer done()

					defer func() {
						_ = copiedCfg.inngest.Process.Kill()
					}()

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
