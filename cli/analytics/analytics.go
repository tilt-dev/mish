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
const contentType = "Content-Type"
const contentTypeJson = "application/json"
const statsTimeout = time.Minute

// keys for request to stats server
const (
	keyName     = "name"
	keyUser     = "user"
	keyDuration = "duration"
)

var cli = &http.Client{Timeout: statsTimeout}

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
	Cli     *http.Client
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
	return newRemoteAnalytics(cli, statsEndpt, getUserId(), optedIn)
}

func newRemoteAnalytics(cli *http.Client, url, userId string, optedIn bool) *RemoteAnalytics {
	return &RemoteAnalytics{Cli: cli, Url: url, UserId: userId, OptedIn: optedIn}
}

func (a *RemoteAnalytics) baseReqBody(name string, tags map[string]string) map[string]interface{} {
	req := map[string]interface{}{keyName: name, keyUser: a.UserId}
	for k, v := range tags {
		req[k] = v
	}
	return req
}

func (a *RemoteAnalytics) makeReq(reqBody map[string]interface{}) (*http.Request, error) {
	j, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %v\n", err)
	}
	reader := bytes.NewReader(j)

	req, err := http.NewRequest(http.MethodPost, a.Url, reader)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %v\n", err)
	}
	req.Header.Add(contentType, contentTypeJson)

	return req, nil
}

func (a *RemoteAnalytics) Count(name string, tags map[string]string, n int) {
	if !a.OptedIn {
		return
	}

	go a.count(name, tags, n)
}

func (a *RemoteAnalytics) count(name string, tags map[string]string, n int) {
	req, err := a.countReq(name, tags, n)
	if err != nil {
		// Stat reporter can't return errs, just print it.
		fmt.Fprintf(os.Stderr, "[analytics] %v\n", err)
		return
	}

	resp, err := a.Cli.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[analytics] http.Post: %v\n", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "[analytics] http.Post returned status: %s\n", resp.Status)
	}
}

func (a *RemoteAnalytics) countReq(name string, tags map[string]string, n int) (*http.Request, error) {
	// TODO: include n
	return a.makeReq(a.baseReqBody(name, tags))
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

	go a.timer(name, dur, tags)

}
func (a *RemoteAnalytics) timer(name string, dur time.Duration, tags map[string]string) {
	req, err := a.timerReq(name, dur, tags)
	if err != nil {
		// Stat reporter can't return errs, just print it.
		fmt.Fprintf(os.Stderr, "[analytics] %v\n", err)
		return
	}

	resp, err := a.Cli.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[analytics] http.Post: %v\n", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "[analytics] http.Post returned status: %s\n", resp.Status)
	}

}

func (a *RemoteAnalytics) timerReq(name string, dur time.Duration, tags map[string]string) (*http.Request, error) {
	reqBody := a.baseReqBody(name, tags)
	reqBody[keyDuration] = dur
	return a.makeReq(reqBody)
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
