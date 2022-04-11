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
	progressRegexp = regexp.MustCompile(`\[.*(\d)/(\d)\] (.+)`)
)

type Artifact struct {
	Hash string
	Tag  string
}

type BuildOpts struct {
	Path string
	Tag  string
	Args []string

	Platform string
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

func (b *Builder) Error() error {
	if b.stderr.err != nil {
		// This is the most important, diagnostic error from the
		// build output itself.
		return b.stderr.err
	}
	if b.err != nil {
		// This is a process error.
		return b.err
	}
	return nil
}

func (b Builder) Progress() float64 {
	progress := b.stderr.Progress()
	if progress >= 100 && !b.done {
		// This is "technically complete" in building, but we're still
		// exporting the image.  Return 99, which is waiting for export.
		// We don't know how long this will take.
		return 95
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

	// Some users won't have installed the correct QEMU binaries for cross-compilation.
	// We want to check if that's the case here.
	if err := verifyBuildx(opts); err != nil {
		return nil, err
	}

	builder := &Builder{
		stderr: newProgressReader(),
		cmd:    exec.Command(path, createBuildCommand(opts)...),
	}
	builder.cmd.Stderr = builder.stderr
	builder.cmd.Stdout = builder.stderr

	return builder, nil
}

func verifyBuildx(o BuildOpts) error {
	cmd := exec.Command("docker", "buildx", "ls")
	byt, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to verify buildx platform support: %w", err)
	}

	if o.Platform == "linux/amd64" {
		if !bytes.Contains(byt, []byte("linux/amd64")) {
			return fmt.Errorf("You don't have buildx x86 compilation support enabled.  To install, run:\n\tdocker run --privileged --rm tonistiigi/binfmt --install amd64")
		}
	}
	return nil
}

func createBuildCommand(o BuildOpts) []string {
	defaults := []string{"buildx", "build", "--load"}
	if o.Platform != "" {
		defaults = append(defaults, "--platform", o.Platform)
	}
	a := append(defaults, o.Args...)
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
	// error records an "error: " outputs from Docker.
	err error
}

func (p *progressReader) Write(byt []byte) (n int, err error) {
	matches := progressRegexp.FindAllStringSubmatch(string(byt), -1)
	if len(matches) > 0 {
		// Take the last match
		match := matches[len(matches)-1]
		if len(match) == 4 {
			p.total, _ = strconv.Atoi(match[2])
			p.current, _ = strconv.Atoi(match[1])
			p.status = fmt.Sprintf("Building step %d of your image", p.current)

		}
		if p.current == p.total {
			p.status = "Exporting image..."
		}
	}

	if strings.Contains(string(byt), "error: ") {
		// If there are errors, set the error here.
		p.err = fmt.Errorf(string(byt))
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
