package sysctl

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	key := "kern.maxfiles"
	v, err := Get(key)
	fmt.Println(v)
	if err != nil {
		t.Errorf("Could not read key %s. err: %s", key, err.Error())
	}
}
