package sysctl

import (
	"testing"
)

func TestGet(t *testing.T) {
	key := "kernel.hostname"
	_, err := Get(key)
	if err != nil {
		t.Errorf("Could not read key from sysctl %s", key)
	}
}
