package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	progressRegexp = regexp.MustCompile(`\[(\d)/(\d)\] (.+)`)
)

type Artifact struct {
	Hash string
	Tag  string
}

type BuildOpts struct {
	Path string
	Tag  string
	Args []string
}

type Builder struct {
	cmd    *exec.Cmd
	stderr *progressReader

	done bool
	err  error
}

func (b *Builder) Start() error {
	err := b.cmd.Start()
	go func() {
		b.err = b.cmd.Wait()
		b.done = true
	}()
	return err
}

func (b *Builder) Wait() error {
	err := b.cmd.Wait()
	b.done = true
	b.err = err
	return err
}

func (b Builder) Done() bool {
	return b.done
}

func (b *Builder) Run() error {
	err := b.cmd.Run()
	b.done = true
	b.err = err
	return err
}

func (b Builder) Progress() float64 {
	progress := b.stderr.Progress()
	if progress == 100 && !b.done {
		// This is "technically complete" in building, but we're still
		// exporting the image.  Return 99, which is waiting for export.
		// We don't know how long this will take.
		return 99
	}
	return progress
}

func (b Builder) ProgressText() string {
	return b.stderr.status
}

// Output returns the last N lines of output from stderr
func (b Builder) Output(n int) string {
	if n <= 0 {
		return b.stderr.buf.String()
	}
	str := b.stderr.buf.String()
	parts := []string{}
	for _, p := range strings.Split(str, "\n") {
		if strings.TrimSpace(p) == "" {
			continue
		}
		parts = append(parts, p)
	}
	if len(parts) < n {
		return str
	}
	return strings.Join(parts[len(parts)-n:], "\n")
}

func NewBuilder(ctx context.Context, opts BuildOpts) (*Builder, error) {
	if opts.Tag != "" {
		opts.Args = append(opts.Args, "-t", opts.Tag)
	}
	if opts.Path != "" {
		opts.Args = append(opts.Args, opts.Path)
	}

	path, err := exec.LookPath("docker")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("Docker was not found")
		}
		return nil, err
	}

	builder := &Builder{
		stderr: newProgressReader(),
		cmd:    exec.Command(path, createBuildCommand(opts.Args)...),
	}
	builder.cmd.Stderr = builder.stderr
	builder.cmd.Stdout = builder.stderr

	return builder, nil
}

func createBuildCommand(args []string) []string {
	a := append([]string{"buildx", "build", "--load"}, args...)
	return a
}

func newProgressReader() *progressReader {
	return &progressReader{
		buf:    &bytes.Buffer{},
		status: "Preparing build...",
	}
}

type progressReader struct {
	buf     *bytes.Buffer
	total   int
	current int
	status  string
}

func (p *progressReader) Write(byt []byte) (n int, err error) {
	matches := progressRegexp.FindAllStringSubmatch(string(byt), -1)
	if len(matches) > 0 {
		// Take the last match
		match := matches[len(matches)-1]
		if len(match) == 4 {
			p.total, _ = strconv.Atoi(match[1])
			p.current, _ = strconv.Atoi(match[2])
			p.status = fmt.Sprintf("Building step %s of your image", match[2])

		}
		if p.current == p.total {
			p.status = "Exporting image..."
		}
	}
	return p.buf.Write(byt)
}

func (p *progressReader) Read(byt []byte) (n int, err error) {
	return p.buf.Read(byt)
}

func (p *progressReader) Progress() float64 {
	if p.current == 0 {
		return 0
	}
	return (float64(p.current) / float64(p.total)) * float64(100)
}
