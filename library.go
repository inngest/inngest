package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const basePath = "./library/library/"

// main parses the library repo and produces a JSON file for all examples.
func main() {
	_ = os.RemoveAll("./library")

	buf := &bytes.Buffer{}
	cmd := exec.Command("git", "clone", "--depth", "1", "git@github.com:inngest/library.git")
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		fmt.Printf("error: %d\noutput: %s\n", err.Error(), buf.String())
		os.Exit(1)
	}

	p := &processor{}
	if err := p.Run(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	byt, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Now that we have our JSON file, we can use this within our API.
	// Right now, this is bundled into the primary site.  We should move
	// this to an S3 bucket and trigger this on new PRs.
	_ = ioutil.WriteFile("./public/json/library.json", byt, 0600)
}

type processor struct {
	dirs []fs.DirEntry

	items []*Item
}

// Run iterates through all subdirectories within basePath ("./library") and
// processes each item if it contains an example.
func (p *processor) Run() error {
	p.items = []*Item{}

	file, err := os.Open(basePath)
	if err != nil {
		return err
	}
	dirs, err := file.ReadDir(0)
	if err != nil {
		return fmt.Errorf("error listing library")
	}

	for _, dir := range dirs {
		item, err := p.handle(dir)
		if err != nil {
			return err
		}
		p.items = append(p.items, item)
	}

	return nil
}

func (p *processor) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.items)
}

func (p *processor) handle(dir fs.DirEntry) (*Item, error) {
	dirname := basePath + dir.Name()
	file, err := os.Open(dirname + "/README.md")
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	defer file.Close()
	byt, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Split the README.md by "---" separators such that we can parse the JSON.
	splits := bytes.Split(byt, []byte("---"))
	if len(splits) != 3 {
		return nil, fmt.Errorf("invalid README format")
	}

	item := &Item{}
	err = json.Unmarshal(splits[1], item)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON within %s", dir.Name())
	}

	// TODO: Ensure workflow is valid.
	workflow, err := p.parseSource(dir, item.Source)
	if err != nil {
		return nil, fmt.Errorf("unable to read workflow in %s: %w", dir.Name(), err)
	}

	item.Description = string(splits[2])
	item.Workflow = string(workflow)
	item.Source = ""

	return item, nil
}

func (p *processor) parseSource(dir fs.DirEntry, source string) ([]byte, error) {
	// Open the workflow cue file.
	// If this contains an HTTP link, download that file up to 32KB
	if !strings.HasPrefix(source, "https://") {
		source := basePath + dir.Name() + "/" + filepath.Clean(source)
		return ioutil.ReadFile(source)
	}

	// Right now, workflows are a single file.
	resp, err := http.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 32*1024))
}

type Item struct {
	Title       string   `json:"title"`
	Subtitle    string   `json:"subtitle"`
	Tags        []string `json:"tags"`
	Source      string   `json:"source,omitempty"`
	Description string   `json:"description"`
	Workflow    string   `json:"workflow"`
}
