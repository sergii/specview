package watch

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type fileState struct {
	Size    int64
	ModTime int64
	Mode    fs.FileMode
}

type Watcher struct {
	done chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

func New(root string, onChange func()) (*Watcher, error) {
	initial, err := snapshot(root)
	if err != nil {
		return nil, err
	}

	w := &Watcher{done: make(chan struct{})}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		previous := initial
		for {
			select {
			case <-w.done:
				return
			case <-ticker.C:
				current, err := snapshot(root)
				if err != nil {
					continue
				}
				if !equal(previous, current) {
					previous = current
					onChange()
				}
			}
		}
	}()
	return w, nil
}

func (w *Watcher) Close() error {
	w.once.Do(func() {
		close(w.done)
		w.wg.Wait()
	})
	return nil
}

func snapshot(root string) (map[string]fileState, error) {
	states := make(map[string]fileState)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		states[path] = fileState{
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			Mode:    info.Mode(),
		}
		return nil
	})
	if os.IsNotExist(err) {
		return states, nil
	}
	return states, err
}

func equal(a, b map[string]fileState) bool {
	if len(a) != len(b) {
		return false
	}
	for path, state := range a {
		if b[path] != state {
			return false
		}
	}
	return true
}
