package gotesplit

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

type cmdRegexp struct{}

func (c *cmdRegexp) run(ctx context.Context, argv []string, outStream io.Writer, errStream io.Writer) error {
	// FIXME:
	if len(argv) < 3 {
		return fmt.Errorf("not enough arguments")
	}
	pkgs := argv[2:]
	total, err := strconv.Atoi(argv[0])
	if err != nil {
		return fmt.Errorf("invalid total: %s", err)
	}
	idx, err := strconv.Atoi(argv[1])
	if err != nil {
		return fmt.Errorf("invalid index: %s", err)
	}

	str, err := getOut(pkgs, detectTags(argv), detectRace(argv), total, idx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(outStream, str)
	return err
}

func getOut(pkgs []string, tags string, withRace bool, total, idx int) (string, error) {
	if total < 1 {
		return "", fmt.Errorf("invalid total: %d", total)
	}
	if idx >= total {
		return "", fmt.Errorf("index shoud be between 0 to total-1, but: %d (total:%d)", idx, total)
	}
	testLists, err := getTestListsFromPkgs(pkgs, tags, withRace)
	if err != nil {
		return "", err
	}
	var list []string
	if len(testLists) > 0 {
		list = testLists[0].list
	}
	testNum := len(list)
	minMemberPerGroup := testNum / total
	mod := testNum % total
	getOffset := func(i int) int {
		return minMemberPerGroup*i + int(math.Min(float64(i), float64(mod)))
	}
	from := getOffset(idx)
	to := getOffset(idx + 1)
	s := list[from:to]

	if len(s) == 0 {
		return "0^", nil
	}
	return "^(?:" + strings.Join(list[from:to], "|") + ")$", nil
}
