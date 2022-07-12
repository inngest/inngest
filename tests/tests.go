package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
)

func main() {
	// 1. Read configs
	// 2. For each config, set up the service
	// 3. For each function, set up the function
	// 4. Run the function, assert output.
	ctx := context.Background()
	if err := do(ctx); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func do(ctx context.Context) error {
	cfgs, err := parseConfigs(ctx)
	if err != nil {
		return err
	}

	fns, err := parseFns(ctx)
	if err != nil {
		return err
	}

	var errs error

	for _, cfg := range cfgs {
		fmt.Println("\n**Running", filepath.Base(cfg.dir)+"**\n")
		cmdCtx, done := context.WithCancel(ctx)
		if err := cfg.Up(cmdCtx); err != nil {
			done()
			errs = multierror.Append(
				errs,
				fmt.Errorf("%s failed: %w", filepath.Base(cfg.dir), err),
			)
			continue
		}

		// Run each test.
		eg := &errgroup.Group{}
		for _, item := range fns {
			fn := item
			eg.Go(func() error {
				return fn.Test(ctx, *cfg)
			})
		}

		if err := eg.Wait(); err != nil {
			done()
			errs = multierror.Append(
				errs,
				fmt.Errorf("%s failed: %w", filepath.Base(cfg.dir), err),
			)

			// Carry on for the next config test.
			continue
		}
		fmt.Println("\n**Finished", filepath.Base(cfg.dir)+"**\n")

		done()
		_ = cfg.Wait()
	}

	return errs
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
