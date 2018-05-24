package io

import (
	"bytes"
	"io"
)

type replaceWriter struct {
	delegate    io.Writer
	original    []byte
	replacement []byte
}

func NewReplaceWriter(delegate io.Writer, original, replacement []byte) io.Writer {
	if delegate == nil {
		panic("Missing delegate")
	}
	return &replaceWriter{delegate: delegate, original: original, replacement: replacement}
}

// TODO(nick): This implementation is not great, but works well for the common cases
// we care about. A more robust implementation would either:
// 1) Buffer input (e.g., based on line terminators), so that the Replace()
//    wouldn't miss "split" directory names, or
// 2) Wait until all the output has been written, and do the replacement somewhere downstream
//    (which would add API complexity to ship the Replace parameters downstream).
func (w *replaceWriter) Write(p []byte) (int, error) {
	p2 := bytes.Replace(p, w.original, w.replacement, -1)
	n, err := w.delegate.Write(p2)
	return (len(p) - (len(p2) - n)), err
}
