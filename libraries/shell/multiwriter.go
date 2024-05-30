package shell

import (
	"io"
	"sync"
)

// MultiWriteCloser is an io.WriteCloser that duplicates its writes and closes to all the provided io.WriteClosers.
type MultiWriteCloser struct {
	writers []io.WriteCloser
}

// NewMultiWriteCloser creates a new MultiWriteCloser.
func NewMultiWriteCloser(writers ...io.WriteCloser) *MultiWriteCloser {
	return &MultiWriteCloser{writers: writers}
}

// Write writes p to all the underlying writers concurrently.
func (mwc *MultiWriteCloser) Write(p []byte) (int, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var writeErr error

	wg.Add(len(mwc.writers))
	for _, w := range mwc.writers {
		go func(w io.WriteCloser) {
			defer wg.Done()
			_, err := w.Write(p)
			if err != nil {
				mu.Lock()
				writeErr = err
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	if writeErr != nil {
		return 0, writeErr
	}
	return len(p), nil
}

// Close closes all the underlying writers.
func (mwc *MultiWriteCloser) Close() error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var closeErr error

	wg.Add(len(mwc.writers))
	for _, w := range mwc.writers {
		go func(w io.WriteCloser) {
			defer wg.Done()
			if err := w.Close(); err != nil {
				mu.Lock()
				closeErr = err
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	return closeErr
}
