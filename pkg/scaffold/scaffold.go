package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/mitchellh/go-homedir"
)

var (
	InngestRoot, _ = homedir.Expand("~/.config/inngest")
	// CacheDir is the directory in which we store the scaffold cache
	CacheDir, _ = homedir.Expand("~/.config/inngest/scaffolds")
	RepoURL     = "https://github.com/inngest/scaffolds.git"
)

type Mapping struct {
	Languages map[string][]Template
}

func Parse(ctx context.Context) (*Mapping, error) {
	return parse(ctx, os.DirFS(CacheDir))
}

func parse(ctx context.Context, dirfs fs.FS) (*Mapping, error) {
	m := &Mapping{
		Languages: map[string][]Template{},
	}

	err := fs.WalkDir(dirfs, ".", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}

		// Ignore .git files and only parse config.json
		if strings.HasPrefix(path, ".git") || !strings.HasSuffix(path, "/config.json") {
			return nil
		}

		parts := strings.SplitN(path, "/", 3)
		if len(parts) != 3 {
			return fmt.Errorf("unexpected path structure in scaffold.  expected '$language/$name/config.json'.")
		}

		file, err := dirfs.Open(path)
		if err != nil {
			return err
		}

		byt, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		t := Template{}
		if err := json.Unmarshal(byt, &t); err != nil {
			return err
		}

		if t.Name == "" {
			return nil
		}

		// Add the data dir as the root.
		t.root = filepath.Join(CacheDir, filepath.Join(".", parts[0], parts[1], "data"))

		language := parts[0]
		m.Languages[language] = append(m.Languages[language], t)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return m, nil
}

// UpdateCache updates the cache of scaffolds from our repository.
func UpdateCache(ctx context.Context) error {
	// Set a timeout of

	_, err := os.Stat(CacheDir)
	if os.IsNotExist(err) {
		return clone(ctx)
	}
	return update(ctx)
}

func update(ctx context.Context) error {
	repo, err := git.PlainOpen(CacheDir)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return err
}

// clone clones the scaffold directory into
func clone(ctx context.Context) error {
	if err := os.MkdirAll(InngestRoot, 0755); err != nil {
		return fmt.Errorf("unable to make cache dir: %w", err)
	}
	_, err := git.PlainClone(CacheDir, false, &git.CloneOptions{
		URL:      RepoURL,
		Progress: nil,
	})
	return err
}
