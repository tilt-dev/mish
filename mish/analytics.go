package mish

import (
	"time"

	"github.com/windmilleng/mish/cli/analytics"
)

func initAnalytics() (analytics.Analytics, error) {
	a, _, err := analytics.Init("mish")
	if err != nil {
		return nil, err
	}

	mishlytics = &mishAnalytics{}

	w, err := a.Register("init", analytics.Nil)
	if err != nil {
		return nil, err
	}
	mishlytics.init = analytics.NewStringWriter(w)

	w, err = a.Register("runs", analytics.Nil)
	if err != nil {
		return nil, err
	}
	mishlytics.runs = &runEventWriter{del: w}

	w, err = a.Register("errors", analytics.Nil)
	if err != nil {
		return nil, err
	}
	mishlytics.errs = analytics.NewErrorWriter(w)

	return a, nil
}

type mishAnalytics struct {
	init analytics.StringWriter
	runs *runEventWriter
	errs analytics.ErrorWriter
}

var mishlytics *mishAnalytics

type runEvent struct {
	runLatency time.Duration
	workflows  int // TODO(dmiller) record this
	length     int
}

type runEventWriter struct {
	del analytics.AnyWriter
}

func (w *runEventWriter) Write(e runEvent) {
	w.del.Write(e)
}
