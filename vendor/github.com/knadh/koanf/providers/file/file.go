// Package file implements a koanf.Provider that reads raw bytes
// from files on disk to be used with a koanf.Parser to parse
// into conf maps.
package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

// File implements a File provider.
type File struct {
	path string
	w    *fsnotify.Watcher

	// Using Go 1.18 atomic functions for backwards compatibility.
	isWatching  uint32
	isUnwatched uint32
}

// Provider returns a file provider.
func Provider(path string) *File {
	return &File{path: filepath.Clean(path)}
}

// ReadBytes reads the contents of a file on disk and returns the bytes.
func (f *File) ReadBytes() ([]byte, error) {
	return os.ReadFile(f.path)
}

// Read is not supported by the file provider.
func (f *File) Read() (map[string]interface{}, error) {
	return nil, errors.New("file provider does not support this method")
}

// Watch watches the file and triggers a callback when it changes. It is a
// blocking function that internally spawns a goroutine to watch for changes.
func (f *File) Watch(cb func(event interface{}, err error)) error {
	// If a watcher already exists, return an error.
	if atomic.LoadUint32(&f.isWatching) == 1 {
		return errors.New("file is already being watched")
	}

	// Resolve symlinks and save the original path so that changes to symlinks
	// can be detected.
	realPath, err := filepath.EvalSymlinks(f.path)
	if err != nil {
		return err
	}
	realPath = filepath.Clean(realPath)

	// Although only a single file is being watched, fsnotify has to watch
	// the whole parent directory to pick up all events such as symlink changes.
	fDir, _ := filepath.Split(f.path)

	f.w, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	atomic.StoreUint32(&f.isWatching, 1)

	var (
		lastEvent     string
		lastEventTime time.Time
	)

	go func() {
	loop:
		for {
			select {
			case event, ok := <-f.w.Events:
				if !ok {
					// Only throw an error if it was not an explicit unwatch.
					if atomic.LoadUint32(&f.isUnwatched) == 0 {
						cb(nil, errors.New("fsnotify watch channel closed"))
					}

					break loop
				}

				// Use a simple timer to buffer events as certain events fire
				// multiple times on some platforms.
				if event.String() == lastEvent && time.Since(lastEventTime) < time.Millisecond*5 {
					continue
				}
				lastEvent = event.String()
				lastEventTime = time.Now()

				evFile := filepath.Clean(event.Name)

				// Resolve symlink to get the real path, in case the symlink's
				// target has changed.
				curPath, err := filepath.EvalSymlinks(f.path)
				if err != nil {
					cb(nil, err)
					break loop
				}
				curPath = filepath.Clean(curPath)

				onWatchedFile := evFile == realPath || evFile == f.path

				// Since the event is triggered on a directory, is this
				// a create or write on the file being watched?
				//
				// Or has the real path of the file being watched changed?
				//
				// If either of the above are true, trigger the callback.
				if event.Has(fsnotify.Create|fsnotify.Write) && (onWatchedFile ||
					(curPath != "" && curPath != realPath)) {
					realPath = curPath

					// Trigger event.
					cb(nil, nil)
				} else if onWatchedFile && event.Has(fsnotify.Remove) {
					cb(nil, fmt.Errorf("file %s was removed", event.Name))
					break loop
				}

			// There's an error.
			case err, ok := <-f.w.Errors:
				if !ok {
					// Only throw an error if it was not an explicit unwatch.
					if atomic.LoadUint32(&f.isUnwatched) == 0 {
						cb(nil, errors.New("fsnotify err channel closed"))
					}

					break loop
				}

				// Pass the error to the callback.
				cb(nil, err)
				break loop
			}
		}

		atomic.StoreUint32(&f.isWatching, 0)
		atomic.StoreUint32(&f.isUnwatched, 0)
		f.w.Close()
	}()

	// Watch the directory for changes.
	return f.w.Add(fDir)
}

// Unwatch stops watching the files and closes fsnotify watcher.
func (f *File) Unwatch() error {
	atomic.StoreUint32(&f.isUnwatched, 1)
	return f.w.Close()
}
