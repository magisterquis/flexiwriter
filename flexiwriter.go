// Package flexiwriter is like io.MultiWriter, but allows for dynamic addition
// and removal of writers.
package flexiwriter

/*
 * flexiwriter.go
 * Dynamic io.MultiWriter
 * By J. Stuart McMurray
 * Created 20220526
 * Last Modified 20220528
 */

import (
	"io"
	"sync"
)

/* singleWriter is a single underlying writer in a Writer. */
type singleWriter struct {
	l   sync.Mutex
	ech chan<- error
	w   io.Writer
}

// Writer is an io.WriteCloser which writes to multiple sub-writers.
type Writer struct {
	l    sync.Mutex
	ws   map[*singleWriter]struct{}
	done bool
}

// New returns a new Writer, ready for use.  The returned writer's Add method
// must be called to add io.Writers.
func New() *Writer { return &Writer{ws: make(map[*singleWriter]struct{})} }

// Add adds an io.Witer to w, such that all writes to w will be written to the
// added io.Writer.  The first error encountered when writing to the io.Writer
// will be sent to the returned channel and the io.Writer will be removed from
// w.  This channel is buffered; it may be discarded without channel leakage.
// The returned function may be called to remove the io.Writer from w, send nil
// to err, and close err.
func (w *Writer) Add(child io.Writer) (remove func(), err <-chan error) {
	w.l.Lock()
	defer w.l.Unlock()
	ech := make(chan error, 1)
	sw := &singleWriter{
		ech: ech,
		w:   child,
	}
	w.ws[sw] = struct{}{}
	return func() {
		w.l.Lock()
		defer w.l.Unlock()
		w.delete(sw, nil)
	}, ech
}

// Write writes to all of the io.Writers Add'd to w.  If a write encounters
// an error, the error is sent to the channel returned from Add and that
// io.Writer is removed from w.  The error returned from Write is always nil.
// Write blocks and blocks other calls to w's methods until all the underlying
// writing has finished.  The returned int is always len(p).
func (w *Writer) Write(p []byte) (int, error) {
	w.l.Lock()
	defer w.l.Unlock()
	var wg sync.WaitGroup
	for cw := range w.ws {
		wg.Add(1)
		go func(sw *singleWriter) {
			defer wg.Done()
			if _, err := sw.w.Write(p); nil != err {
				w.delete(sw, err)
			}
		}(cw)
	}
	wg.Wait()

	return len(p), nil
}

// Close prevents further writes to w and closes all of its underlying writers
// which implement io.Closer.  It always returns nil.
func (w *Writer) Close() error {
	w.l.Lock()
	defer w.l.Unlock()
	/* Don't double-close. */
	if w.done {
		return nil
	}
	for cw := range w.ws {
		if c, ok := cw.w.(io.Closer); ok {
			c.Close()
		}
		w.delete(cw, nil)
	}
	return nil
}

/* delete sends err to c.ech if it's the first error sent, closes c.ech, and
removes c from w.  w.l must be held during the call to delete.  w.l must be
held during the call to delete. */
func (w *Writer) delete(sw *singleWriter, err error) {
	sw.l.Lock()
	defer sw.l.Unlock()
	delete(w.ws, sw)
	if nil != sw.ech {
		sw.ech <- err
		close(sw.ech)
		sw.ech = nil
	}
}
