package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
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
}

func (b Builder) Start() error {
	return b.cmd.Start()
}

func (b Builder) Wait() error {
	return b.cmd.Wait()
}

func (b Builder) Run() error {
	return b.cmd.Run()
}

func (b Builder) Progress() float64 {
	return b.stderr.Progress()
}

func (b Builder) ProgressText() string {
	return b.stderr.text
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
		buf:  &bytes.Buffer{},
		text: "Preparing build...",
	}
}

type progressReader struct {
	buf     *bytes.Buffer
	total   int
	current int
	text    string
}

func (p *progressReader) Write(byt []byte) (n int, err error) {
	matches := progressRegexp.FindAllStringSubmatch(string(byt), -1)
	if len(matches) > 0 {
		// Take the last match
		match := matches[len(matches)-1]
		if len(match) == 4 {
			p.total, _ = strconv.Atoi(match[1])
			p.current, _ = strconv.Atoi(match[2])
			p.text = match[3]

		}
		if p.current == p.total {
			p.text = "Exporting image..."
		}
		p.total++
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
