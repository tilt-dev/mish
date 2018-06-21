package analytics

import (
	"fmt"
	"os"
	"strings"

	"github.com/windmilleng/mish/cli/dirs"
)

type AnalyticsOpt int

const (
	AnalyticsOptDefault AnalyticsOpt = iota
	AnalyticsOptOut
	AnalyticsOptIn
)

const optInByDefault = false

var choices = map[AnalyticsOpt]string{
	AnalyticsOptDefault: "default",
	AnalyticsOptOut:     "opt-out",
	AnalyticsOptIn:      "opt-in",
}

func OptStatus() (string, error) {
	txt, err := readChoiceFile()
	if err != nil {
		return txt, err
	}
	choice := AnalyticsOptDefault

	switch txt {
	case choices[AnalyticsOptIn]:
		choice = AnalyticsOptIn
	case choices[AnalyticsOptOut]:
		choice = AnalyticsOptOut
	}

	return choices[choice], nil
}

func SetOpt(c string) error {
	choice := AnalyticsOptDefault
	for k, v := range choices {
		if v == c {
			choice = k
		}
	}

	c = choices[choice] // make sure we're using a valid string

	d, err := dirs.UseWindmillDir()
	if err != nil {
		return err
	}

	if err = d.WriteFile(choiceFile, c); err != nil {
		return err
	}

	return nil
}

func readChoiceFile() (string, error) {
	d, err := dirs.UseWindmillDir()
	if err != nil {
		return "", err
	}

	txt, err := d.ReadFile(choiceFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		txt = ""
	}

	return strings.TrimSpace(txt), nil
}

func optedIn() bool {
	txt, err := readChoiceFile()
	if txt != "" {
		switch txt {
		case choices[AnalyticsOptOut]:
			return false
		case choices[AnalyticsOptIn]:
			return true
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "analytics.optedIn: %v\n", err)
	}
	return optInByDefault
}
