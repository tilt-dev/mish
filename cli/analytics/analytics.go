package analytics

import (
	"github.com/spf13/cobra"
)

func Init() (*cobra.Command, error) {
	c, err := initCLI()
	return c, err
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
