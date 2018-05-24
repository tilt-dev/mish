package sysctl

import (
	"fmt"
	"io/ioutil"
	"strings"
)

const sysctlDir = "/proc/sys/"

func Get(name string) (string, error) {
	path := sysctlDir + strings.Replace(name, ".", "/", -1)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not find key %s", name)
	}
	return strings.TrimSpace(string(data)), nil
}
