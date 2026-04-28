// Package file implements a koanf.Provider that reads raw bytes
// from files on disk to be used with a koanf.Parser to parse
// into conf maps.
package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// File implements a File provider.
type File struct {
	path string
	w    *fsnotify.Watcher

	// Mutex to protect concurrent access to watcher state
	mu         sync.Mutex
	isWatching bool
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
func (f *File) Read() (map[string]any, error) {
	return nil, errors.New("file provider does not support this method")
}

// Watch watches the file and triggers a callback when it changes. It is a
// blocking function that internally spawns a goroutine to watch for changes.
func (f *File) Watch(cb func(event any, err error)) error {
	f.mu.Lock()

	// If a watcher already exists, return an error.
	if f.isWatching {
		f.mu.Unlock()
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
		f.mu.Unlock()
		return err
	}

	f.isWatching = true

	// Set up the directory watch before releasing the lock
	err = f.w.Add(fDir)
	if err != nil {
		f.w.Close()
		f.w = nil
		f.isWatching = false
		f.mu.Unlock()
		return err
	}

	// Release the lock before spawning goroutine
	f.mu.Unlock()

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
					// Only throw an error if we were still supposed to be watching.
					f.mu.Lock()
					stillWatching := f.isWatching
					f.mu.Unlock()

					if stillWatching {
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
					cb(event, nil)
				} else if onWatchedFile && event.Has(fsnotify.Remove) {
					cb(nil, fmt.Errorf("file %s was removed", event.Name))
					break loop
				}

			// There's an error.
			case err, ok := <-f.w.Errors:
				if !ok {
					// Only throw an error if we were still supposed to be watching.
					f.mu.Lock()
					stillWatching := f.isWatching
					f.mu.Unlock()

					if stillWatching {
						cb(nil, errors.New("fsnotify err channel closed"))
					}

					break loop
				}

				// Pass the error to the callback.
				cb(nil, err)
				break loop
			}
		}

		f.mu.Lock()
		f.isWatching = false
		if f.w != nil {
			f.w.Close()
			f.w = nil
		}
		f.mu.Unlock()
	}()

	return nil
}

// Unwatch stops watching the files and closes fsnotify watcher.
func (f *File) Unwatch() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isWatching {
		return nil // Already unwatched
	}

	f.isWatching = false
	if f.w != nil {
		// Close the watcher to signal the goroutine to stop
		// The goroutine will handle setting f.w = nil
		return f.w.Close()
	}
	// This state should ideally never be reached - it indicates a bug in the synchronization logic
	return errors.New("file watcher is in an inconsistent state: isWatching is true but watcher is nil")
}
