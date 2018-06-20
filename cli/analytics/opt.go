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

var choices = map[AnalyticsOpt]string{
	AnalyticsOptDefault: "default",
	AnalyticsOptOut:     "opt-out",
	AnalyticsOptIn:      "opt-in",
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
