package imagecmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	isatty "github.com/mattn/go-isatty"
)

// isTerminal reports whether the questionnaire can ask questions: stdin and
// stdout are both terminals.
func isTerminal(stdin io.Reader, stdout io.Writer) bool {
	in, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	out, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	terminal := func(f *os.File) bool {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return terminal(in) && terminal(out)
}

// writeAppsFile asks for the settings of one app, like `npm init`, and writes
// inngest.toml in the current directory.
func writeAppsFile(stdin io.Reader, stdout io.Writer) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s not found.  Answer a few questions to create it, or press Ctrl-C to stop.\n", appsFile)
	fmt.Fprintf(stdout, "Press enter to keep the default in parentheses.  `inngest deploy example-config` shows every setting.\n\n")

	q := questions{in: bufio.NewScanner(stdin), out: stdout}
	name := q.ask("App name", defaultAppName(filepath.Base(wd)), func(v string) error {
		if !namePattern.MatchString(v) || v == "default" {
			return errors.New(`use 1 to 63 lowercase letters, digits, '.', '_' or '-', and not "default"`)
		}
		return nil
	})
	mode := q.ask("Run the app with serve or connect", "serve", func(v string) error {
		if v != "serve" && v != "connect" {
			return errors.New("answer serve or connect")
		}
		return nil
	})
	var port, path, replicas string
	if mode == "serve" {
		port = q.ask("Port the app listens on", "8080", uintBetween(1, 65535))
		path = q.ask("Path of the Inngest endpoint", "/api/inngest", func(v string) error {
			if !strings.HasPrefix(v, "/") || strings.ContainsAny(v, "?#") {
				return errors.New("use an absolute path such as /api/inngest, without a query or fragment")
			}
			return nil
		})
	} else {
		replicas = q.ask("Number of workers", "1", uintBetween(1, math.MaxUint32))
	}
	dockerfile := q.ask("Dockerfile", defaultDockerfile, nil)
	if q.err != nil {
		return fmt.Errorf("%s was not written: %w", appsFile, q.err)
	}

	var b strings.Builder
	b.WriteString("# Written by `inngest deploy`.  `inngest deploy example-config` shows every setting.\n")
	fmt.Fprintf(&b, "[apps.%s]\n", tomlKey(name))
	if mode == "serve" {
		fmt.Fprintf(&b, "serve = { port = %s, path = %s }\n", port, strconv.Quote(path))
	} else {
		fmt.Fprintf(&b, "connect = { replicas = %s }\n", replicas)
	}
	fmt.Fprintf(&b, "dockerfile = %s\n", strconv.Quote(dockerfile))

	// the file must be one that `inngest deploy` accepts.
	if _, err := parseApps(strings.NewReader(b.String())); err != nil {
		return err
	}
	f, err := os.OpenFile(appsFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(b.String()); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "\nWrote %s:\n\n%s\nCheck it, then run `inngest deploy` to deploy %s.\n", appsFile, b.String(), name)
	return nil
}

// questions reads one answer per line.  after the first read error, ask
// returns its default and err holds the error.
type questions struct {
	in  *bufio.Scanner
	out io.Writer
	err error
}

func (q *questions) ask(question, value string, validate func(string) error) string {
	for q.err == nil {
		fmt.Fprintf(q.out, "%s: (%s) ", question, value)
		if !q.in.Scan() {
			q.err = q.in.Err()
			if q.err == nil {
				q.err = io.ErrUnexpectedEOF
			}
			break
		}
		answer := strings.TrimSpace(q.in.Text())
		if answer == "" {
			answer = value
		}
		err := validateText(answer)
		if err == nil && validate != nil {
			err = validate(answer)
		}
		if err != nil {
			fmt.Fprintf(q.out, "%v\n", err)
			continue
		}
		return answer
	}
	return value
}

// validateText rejects text that strconv.Quote does not write as a valid
// TOML string.
func validateText(v string) error {
	if !utf8.ValidString(v) || strings.IndexFunc(v, unicode.IsControl) >= 0 {
		return errors.New("use printable characters only")
	}
	return nil
}

func uintBetween(low, high uint64) func(string) error {
	return func(v string) error {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil || n < low || n > high {
			if high == math.MaxUint32 {
				return fmt.Errorf("use a whole number of at least %d", low)
			}
			return fmt.Errorf("use a number from %d to %d", low, high)
		}
		return nil
	}
}

var invalidNameChars = regexp.MustCompile(`[^a-z0-9._-]+`)

// defaultAppName turns the directory name into an app name.
func defaultAppName(dir string) string {
	name := invalidNameChars.ReplaceAllString(strings.ToLower(dir), "-")
	name = strings.TrimLeft(name, "._-")
	if len(name) > 63 {
		name = name[:63]
	}
	if !namePattern.MatchString(name) || name == "default" {
		return "my-app"
	}
	return name
}

// tomlKey quotes names that contain '.', which a bare TOML key splits.
func tomlKey(name string) string {
	if strings.Contains(name, ".") {
		return strconv.Quote(name)
	}
	return name
}
