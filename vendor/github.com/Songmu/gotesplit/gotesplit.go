package gotesplit

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

const cmdName = "gotesplit"

// Run the gotesplit
func Run(ctx context.Context, argv []string, outStream, errStream io.Writer) error {
	log.SetOutput(errStream)
	fs := flag.NewFlagSet(
		fmt.Sprintf("%s (v%s rev:%s)", cmdName, version, revision), flag.ContinueOnError)
	fs.SetOutput(errStream)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage of %s:
  $ gotesplit [options] [pkgs...] [-- go-test-arguments...]

Description:
  split the testng in Go into a subset and run it

Example:
  $ gotesplit -total=10 -index=0 -- -v -short
  go test -v -short -run ^(?:TestAA|TestBB)$

Options:
`, fs.Name())
		fs.PrintDefaults()
	}
	total := fs.Uint("total", 1, "total number of test splits (CIRCLE_NODE_TOTAL is used if set)")
	index := fs.Uint("index", 0, "zero-based index number of test splits (CIRCLE_NODE_INDEX is used if set)")
	junitDir := fs.String("junit-dir", "", "directory to store test result in JUnit format")
	coverprofileDir := fs.String("coverprofile-dir", ".cover", "temporary directory for collecting coverprofile")
	fs.VisitAll(func(f *flag.Flag) {
		if f.Name == "index" || f.Name == "total" {
			if s := os.Getenv("CIRCLE_NODE_" + strings.ToUpper(f.Name)); s != "" {
				f.Value.Set(s)
			}
		}
	})
	if err := fs.Parse(argv); err != nil {
		return err
	}
	argv = fs.Args()
	if len(argv) > 0 {
		rnr, ok := dispatch[argv[0]]
		if ok {
			return rnr.run(ctx, argv[1:], outStream, errStream)
		}
	}
	return run(ctx, *total, *index, *junitDir, *coverprofileDir, argv, outStream, errStream)
}

func getTestListsFromPkgs(pkgs []string, tags string, withRace bool) ([]testList, error) {
	args := []string{"test", "-list", "."}
	if tags != "" {
		args = append(args, tags)
	}
	if withRace {
		// If -race is specified for test options, add -race to list
		// to prevent compilation from being executed twice.
		args = append(args, "-race")
	}
	args = append(args, pkgs...)
	buf := &bytes.Buffer{}
	c := exec.Command("go", args...)
	c.Stdout = buf
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		buf.WriteTo(os.Stdout)
		return nil, err
	}
	return getTestLists(buf.String()), nil
}

var tagsReg = regexp.MustCompile(`^--?tags(=.*)?$`)

func detectTags(argv []string) string {
	l := len(argv)
	for i := 0; i < l; i++ {
		tags := argv[i]
		m := tagsReg.FindStringSubmatch(tags)
		if len(m) < 2 {
			continue
		}
		if m[1] == "" && i+1 < l {
			tags += "=" + argv[i+1]
		}
		return tags
	}
	return ""
}

func detectRace(argv []string) bool {
	l := len(argv)
	for i := 0; i < l; i++ {
		if argv[i] == "-race" || argv[i] == "--race" {
			return true
		}
	}
	return false
}

type testList struct {
	pkg  string
	list []string
}

func getTestLists(out string) []testList {
	var lists []testList
	var list []string
	for _, v := range strings.Split(out, "\n") {
		if strings.HasPrefix(v, "Test") || strings.HasPrefix(v, "Example") {
			list = append(list, v)
			continue
		}
		if strings.HasPrefix(v, "ok ") {
			stuff := strings.Fields(v)
			if len(stuff) != 3 {
				continue
			}
			sort.Strings(list)
			lists = append(lists, testList{
				pkg:  stuff[1],
				list: list,
			})
			list = nil
		}
	}
	sort.Slice(lists, func(i, j int) bool {
		cmp := len(lists[i].list) - len(lists[j].list)
		if cmp != 0 {
			return cmp < 0
		}
		return strings.Compare(lists[i].pkg, lists[j].pkg) < 0
	})
	return lists
}
