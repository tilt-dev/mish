package sysctl

import "golang.org/x/sys/unix"

func Get(name string) (string, error) {
	return unix.Sysctl(name)
}
