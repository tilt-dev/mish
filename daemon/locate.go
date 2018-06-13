package daemon

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/windmilleng/mish/cli/dirs"
)

func LocateSocket() (string, error) {
	dir, err := dirs.GetWindmillDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "socket"), nil
}

func tagPath() (string, error) {
	dir, err := dirs.GetWindmillDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tag.txt"), nil
}

func ReadCurrentTag() (string, error) {
	tagPath, err := tagPath()
	if err != nil {
		return "", err
	}

	contents, err := ioutil.ReadFile(tagPath)
	if os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(contents)), err
}

func WriteCurrentTag(tag string) error {
	tagPath, err := tagPath()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(tagPath, []byte(tag), os.FileMode(0600))
}
