package analytics

import (
	"testing"

	"net/http"

	"time"

	"github.com/windmilleng/mish/stats/server"
	ts "github.com/windmilleng/mish/stats/testing"
)

var testCli = &http.Client{}
var testTagsBar = map[string]string{"foo": "bar"}
var testTagsBaz = map[string]string{"foo": "baz"}

const testEndpt = "/report"

func TestRemoteAnalyticsCount(t *testing.T) {
	f := ts.NewStatsServerFixture(t)
	a := newTestRemoteAnalytics()
	countReq, err := a.countReq("myStat", nil, 1)
	if err != nil {
		t.Fatal("countReq", err)
	}
	f.Serve(countReq)
	f.AssertCode(200)
	sum := f.R.SumForStatCount(a.namespaced("myStat"))
	if sum != 1 {
		t.Fatalf("Expected event count 1. Actual: %d", sum)
	}
}

func TestRemoteAnalyticsCountWithTags(t *testing.T) {
	f := ts.NewStatsServerFixture(t)
	a := newTestRemoteAnalytics()
	countReq, err := a.countReq("myStat", testTagsBar, 1)
	if err != nil {
		t.Fatal("countReq", err)
	}
	f.Serve(countReq)
	f.AssertCode(200)

	countReq, err = a.countReq("myStat", testTagsBaz, 1)
	if err != nil {
		t.Fatal("countReq", err)
	}
	f.Serve(countReq)
	f.AssertCode(200)

	sum := f.R.SumForStatCountWithTags(a.namespaced("myStat"), server.TagsToStrs(testTagsBar))
	if sum != 1 {
		t.Fatalf("Expected event count 1. Actual: %d", sum)
	}
}

func TestRemoteAnalyticsTimer(t *testing.T) {
	f := ts.NewStatsServerFixture(t)
	a := newTestRemoteAnalytics()
	timerReq, err := a.timerReq("myTimer", time.Duration(1), nil)
	if err != nil {
		t.Fatal("timerReq", err)
	}
	f.Serve(timerReq)
	f.AssertCode(200)
	sum := f.R.SumForTimer(a.namespaced("myTimer"))
	if sum != 1 {
		t.Fatalf("Expected duration 1. Actual: %d", sum)
	}
}

func TestRemoteAnalyticsTimerWithTags(t *testing.T) {
	f := ts.NewStatsServerFixture(t)
	a := newTestRemoteAnalytics()
	timerReq, err := a.timerReq("myTimer", time.Duration(1), testTagsBar)
	if err != nil {
		t.Fatal("timerReq", err)
	}
	f.Serve(timerReq)
	f.AssertCode(200)

	timerReq, err = a.timerReq("myTimer", time.Duration(1), testTagsBaz)
	if err != nil {
		t.Fatal("timerReq", err)
	}
	f.Serve(timerReq)
	f.AssertCode(200)

	sum := f.R.SumForTimerWithTags(a.namespaced("myTimer"), server.TagsToStrs(testTagsBar))
	if sum != 1 {
		t.Fatalf("Expected duration 1. Actual: %d", sum)
	}
}

func newTestRemoteAnalytics() *RemoteAnalytics {
	return newRemoteAnalytics(testCli, "test-app", testEndpt, "random-user", true)
}
