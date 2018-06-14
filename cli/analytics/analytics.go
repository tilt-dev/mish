package analytics

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Init() (Analytics, *cobra.Command, error) {
	c, err := initCLI()
	return newMemoryAnalytics(), c, err
}

type UserCollection int

const (
	CollectDefault UserCollection = iota
	CollectNotEvenLocal
	CollectLocal
	CollectUpload
)

var choices = map[UserCollection]string{
	CollectDefault:      "default",
	CollectNotEvenLocal: "not-even-local",
	CollectLocal:        "local",
	CollectUpload:       "upload",
}

type Analytics interface {
	Register(name string, agg Aggregation) (AnyWriter, error)
}

type AnyWriter interface {
	Write(v interface{})
}

type StringWriter interface {
	Write(v string)
}

type ErrorWriter interface {
	Write(v error)
}

type NoopAnyWriter struct {
}

func NewNoopAnyWriter() AnyWriter {
	return &NoopAnyWriter{}
}

func (w *NoopAnyWriter) Write(v interface{}) {}

type DelegatingStringWriter struct {
	del AnyWriter
}

func NewStringWriter(w AnyWriter) StringWriter {
	return &DelegatingStringWriter{del: w}
}

func (w *DelegatingStringWriter) Write(v string) {
	w.del.Write(v)
}

type DelegatingErrorWriter struct {
	del AnyWriter
}

func NewErrorWriter(w AnyWriter) ErrorWriter {
	return &DelegatingErrorWriter{del: w}
}

func (w *DelegatingErrorWriter) Write(v error) {
	w.del.Write(v)
}

type MemoryAnalytics struct {
	registered map[string]bool
}

func newMemoryAnalytics() *MemoryAnalytics {
	return &MemoryAnalytics{
		registered: make(map[string]bool),
	}
}

func (a *MemoryAnalytics) Register(name string, agg Aggregation) (AnyWriter, error) {
	if a.registered[name] {
		return nil, fmt.Errorf("duplicate key %v", name)
	}

	return &NoopAnyWriter{}, nil
}
