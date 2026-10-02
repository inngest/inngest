package image

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

const maxArchiveBytes = 10 << 30
const maxExpandedBytes = 20 << 30

type boundedWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errors.New("image archive exceeds size limit")
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}

func compressedArchive(file *os.File, produce func(io.Writer) error) (string, int64, error) {
	hash := sha256.New()
	gz := gzip.NewWriter(&boundedWriter{writer: io.MultiWriter(file, hash), remaining: maxArchiveBytes})
	if err := produce(gz); err != nil {
		_ = gz.Close()
		return "", 0, err
	}
	if err := gz.Close(); err != nil {
		return "", 0, err
	}
	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), stat.Size(), nil
}

// Dockerfile-specific ignore rules take precedence over .dockerignore. Walk
// excluded directories too: later !rules can include individual descendants.
func contextArchive(ctx context.Context, directory, dockerfile string, out io.Writer) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	var patterns []string
	for _, name := range []string{dockerfile + ".dockerignore", ".dockerignore"} {
		f, err := root.Open(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		patterns, err = ignorefile.ReadAll(io.LimitReader(f, 1<<20))
		_ = f.Close()
		if err != nil {
			return err
		}
		break
	}
	matcher, err := patternmatcher.New(patterns)
	if err != nil {
		return err
	}
	tw := tar.NewWriter(out)
	var total int64
	var files int
	err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(rel)
		ignored, err := matcher.MatchesOrParentMatches(name)
		if err != nil {
			return err
		}
		if ignored {
			return nil
		}
		info, err := root.Lstat(rel)
		if err != nil {
			return err
		}
		link := ""
		if info.Mode()&fs.ModeSymlink != 0 {
			link, err = root.Readlink(rel)
			if err != nil {
				return err
			}
			if filepath.IsAbs(link) || !filepath.IsLocal(filepath.Join(filepath.Dir(rel), link)) {
				return fmt.Errorf("context symlink %s escapes the build context", name)
			}
		} else if !info.Mode().IsRegular() && !info.IsDir() {
			return fmt.Errorf("context contains unsupported special file %s", name)
		}
		files++
		if files > 1_000_000 {
			return errors.New("build context exceeds file limit")
		}
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		header.Name = name
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""
		header.ModTime = time.Unix(0, 0)
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}
		total += header.Size
		if total > maxExpandedBytes {
			return errors.New("build context exceeds expanded size limit")
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := root.Open(rel)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(tw, f, header.Size)
			closeErr := f.Close()
			return errors.Join(copyErr, closeErr)
		}
		return nil
	})
	return errors.Join(err, tw.Close())
}

func dockerfileText(directory, name string) (string, error) {
	if !filepath.IsLocal(name) {
		return "", errors.New("remote Dockerfile must be inside the build context")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	f, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (256<<10)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 256<<10 || strings.ContainsRune(string(data), 0) {
		return "", errors.New("Dockerfile must be at most 256 KiB without NUL bytes")
	}
	return string(data), nil
}
