package io

import (
	"io"
)

// A writer that truncates the data stream after N bytes
type truncateWriter struct {
	delegate  io.Writer
	maxBytes  int
	endMarker []byte
}

// endMarker: A marker to put at the end of the string to indicate that it
// has been truncated.
func NewTruncateWriter(delegate io.Writer, maxBytes int, endMarker []byte) io.Writer {
	if delegate == nil {
		panic("Missing delegate")
	}
	return &truncateWriter{delegate: delegate, maxBytes: maxBytes, endMarker: endMarker}
}

func (w *truncateWriter) Write(p []byte) (int, error) {
	l := len(p)
	maxBytes := w.maxBytes
	if maxBytes <= 0 {
		return l, nil
	}

	w.maxBytes = maxBytes - len(p)

	// This is a little bit tricky because we have to truncate the byte array first.
	// But to fulfill the Writer contract, we have to return l if there's no error.
	if len(p) > maxBytes {
		p = p[0:maxBytes]
	}

	l2, err := w.delegate.Write(p)
	if err != nil {
		return l2, err
	}

	if w.maxBytes <= 0 {
		_, err := w.delegate.Write(w.endMarker)
		if err != nil {
			return l, err
		}
	}
	return l, nil
}
