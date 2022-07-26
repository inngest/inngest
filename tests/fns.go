package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/tests/testdsl"

	_ "github.com/inngest/inngest/tests/fns/basic-single-step"
)

// parseFns reads all functions from "./fns"
func parseFns(ctx context.Context) ([]*Fn, error) {
	fns := []*Fn{}
	abs, _ := filepath.Abs("./fns")
	entries, _ := os.ReadDir("./fns")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		fn := &Fn{dir: filepath.Join(abs, e.Name())}
		if err := fn.Validate(ctx); err != nil {
			return nil, err
		}
		fns = append(fns, fn)
	}
	return fns, nil
}

type Fn struct {
	fn  *function.Function
	dir string
}

func (f *Fn) Validate(ctx context.Context) error {
	config, err := os.ReadFile(filepath.Join(f.dir, "inngest.json"))
	if err != nil {
		return err
	}
	f.fn, err = function.Unmarshal(ctx, config, f.dir)
	if err != nil {
		return err
	}

	if root := testdsl.ForDir(filepath.Base(f.dir)); root == nil {
		return fmt.Errorf("unable to find root test DSL proc for fn: %s", f.dir)
	}
	return nil
}

func (f *Fn) Test(ctx context.Context, c Config) error {
	dirname := filepath.Base(f.dir)
	root := testdsl.ForDir(dirname)

	testdata := &testdsl.TestData{
		Fn:     f.fn,
		Config: c.config,
		Out:    c.out,
	}

	chain := root(ctx)
	for _, f := range chain {
		if err := f(ctx, testdata); err != nil {
			return err
		}
	}

	return nil
}
