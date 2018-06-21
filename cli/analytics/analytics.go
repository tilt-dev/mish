package analytics

import (
	"fmt"

	"time"

	"net/http"

	"encoding/json"

	"bytes"
	"os"

	"github.com/spf13/cobra"
)

const statsEndpt = "http://localhost:8118/report"
const contentTypeJson = "application/json"

// keys for request to stats server
const (
	keyName     = "name"
	keyUser     = "user"
	keyDuration = "duration"
)

func Init() (Analytics, *cobra.Command, error) {

	a := NewRemoteAnalytics()
	c, err := initCLI()
	if err != nil {
		return nil, nil, err
	}

	return a, c, nil
}

type Analytics interface {
	Count(name string, tags map[string]string, n int)
	Incr(name string, tags map[string]string)
	Timer(name string, dur time.Duration, tags map[string]string)
}

type RemoteAnalytics struct {
	Url     string
	UserId  string
	OptedIn bool
}

// getUserHash returns a unique identifier for this user by hashing... something.
func getUserId() string {
	return "getUserId_not_implemented"
}

func NewRemoteAnalytics() *RemoteAnalytics {
	optedIn := optedIn()
	return newRemoteAnalytics(statsEndpt, getUserId(), optedIn)
}

func newRemoteAnalytics(url, userId string, optedIn bool) *RemoteAnalytics {
	return &RemoteAnalytics{Url: url, UserId: userId, OptedIn: optedIn}
}

func (a *RemoteAnalytics) baseReq(name string, tags map[string]string) map[string]interface{} {
	req := map[string]interface{}{keyName: name, keyUser: a.UserId}
	for k, v := range tags {
		req[k] = v
	}
	return req
}

func (a *RemoteAnalytics) postReq(req map[string]interface{}) {
	j, err := json.Marshal(req)
	if err != nil {
		// Stat reporter can't return errs, just print it.
		fmt.Fprintf(os.Stderr, "[analytics] json.Marshal: %v\n", err)
		return
	}
	reader := bytes.NewReader(j)
	resp, err := http.Post(a.Url, contentTypeJson, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[analytics] http.Post: %v\n", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "[analytics] http.Post returned status: %s\n", resp.Status)
	}
}

func (a *RemoteAnalytics) Count(name string, tags map[string]string, n int) {
	if !a.OptedIn {
		return
	}

	// TODO: include n
	req := a.baseReq(name, tags)
	a.postReq(req)

}

func (a *RemoteAnalytics) Incr(name string, tags map[string]string) {
	if !a.OptedIn {
		return
	}
	a.Count(name, tags, 1)
}

func (a *RemoteAnalytics) Timer(name string, dur time.Duration, tags map[string]string) {
	if !a.OptedIn {
		return
	}

	req := a.baseReq(name, tags)
	req[keyDuration] = dur
	a.postReq(req)
}

type MemoryAnalytics struct {
	Counts []CountEvent
	Timers []TimeEvent
}

type CountEvent struct {
	name string
	tags map[string]string
	n    int
}

type TimeEvent struct {
	name string
	tags map[string]string
	dur  time.Duration
}

func NewMemoryAnalytics() *MemoryAnalytics {
	return &MemoryAnalytics{}
}

func (a *MemoryAnalytics) Count(name string, tags map[string]string, n int) {
	a.Counts = append(a.Counts, CountEvent{name: name, tags: tags, n: n})
}

func (a *MemoryAnalytics) Incr(name string, tags map[string]string) {
	a.Count(name, tags, 1)
}

func (a *MemoryAnalytics) Timer(name string, dur time.Duration, tags map[string]string) {
	a.Timers = append(a.Timers, TimeEvent{name: name, dur: dur, tags: tags})
}

var _ Analytics = &RemoteAnalytics{}
var _ Analytics = &MemoryAnalytics{}
