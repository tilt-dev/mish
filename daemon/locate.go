package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

func CurrentHomeDir() (string, error) {
	home := os.Getenv("HOME")
	if home != "" {
		return home, nil
	}

	current, err := user.Current()
	if err != nil {
		return "", err
	}
	return current.HomeDir, nil
}

// WindmillDir returns the root Windmill directory; by default ~/.windmill
func WindmillDir() (string, error) {
	dir := os.Getenv("WMDAEMON_HOME")
	if dir == "" {
		homedir, err := CurrentHomeDir()
		if err != nil {
			return "", err
		}

		if homedir == "" {
			return "", fmt.Errorf("Cannot find home directory; $HOME unset")
		}
		dir = path.Join(homedir, ".windmill")
	}

	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(dir, os.FileMode(0755))
		} else {
			return "", err
		}
	}

	return dir, nil
}

func LocateSocket() (string, error) {
	dir, err := WindmillDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, "socket"), nil
}

func tagPath() (string, error) {
	dir, err := WindmillDir()
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
