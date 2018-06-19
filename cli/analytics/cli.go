package analytics

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/windmilleng/mish/cli/dirs"
)

func analyticsStatus(_ *cobra.Command, args []string) error {
	//fmt.Println("[ for more info: https://windmill.engineering/analytics ]")

	txt, err := readChoiceFile()
	if err != nil {
		return fmt.Errorf("readChoiceFile: %v", err)
	}

	choice := AnalyticsOptDefault

	switch txt {
	case choices[AnalyticsOptOut]:
		choice = AnalyticsOptOut
	case choices[AnalyticsOptIn]:
		choice = AnalyticsOptIn
	}

	fmt.Printf("current collection strategy: %v\n", choices[choice])

	return nil
}

func analyticsOpt(_ *cobra.Command, args []string) (outerErr error) {
	fmt.Printf("choice can be one of {%v, %v}\n", choices[AnalyticsOptIn], choices[AnalyticsOptOut])
	if len(args) == 0 {
		return fmt.Errorf("no choice given; pass it as first arg: <tool> analytics opt <choice>")
	}
	choiceStr := args[0]
	var choice AnalyticsOpt
	for k, v := range choices {
		if v == choiceStr {
			choice = k
		}
	}

	choiceStr = choices[choice] // make sure we're using a valid string

	d, err := dirs.UseWindmillDir()
	if err != nil {
		return err
	}

	if err := d.WriteFile(choiceFile, choiceStr); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "wrote user collection strategy %q to file %v\n", choiceStr, filepath.Join(d.Root(), choiceFile))
	return nil
}

const choiceFile = "analytics/user/choice.txt"

func initCLI() (*cobra.Command, error) {
	analytics := &cobra.Command{
		Use:   "analytics",
		Short: "info and status about windmill analytics",
		RunE:  analyticsStatus,
	}

	opt := &cobra.Command{
		Use:   "opt",
		Short: "opt-in or -out to windmill analytics collection/upload",
		RunE:  analyticsOpt,
	}
	analytics.AddCommand(opt)

	return analytics, nil
}
