package watch

import "github.com/windmilleng/fsnotify"

type wmNotify interface {
	Close() error
	Add(name string) error
	Events() chan fsnotify.Event
	Errors() chan error
}

func NewWatcher() (wmNotify, error) {
	return newWMWatcher()
}
