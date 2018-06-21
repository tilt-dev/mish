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
	choice, err := OptStatus()
	if err != nil {
		return err
	}
	fmt.Printf("current collection strategy: %v\n", choice)

	return nil
}

func analyticsOpt(_ *cobra.Command, args []string) (outerErr error) {
	fmt.Printf("choice can be one of {%v, %v}\n", choices[AnalyticsOptIn], choices[AnalyticsOptOut])
	if len(args) == 0 {
		return fmt.Errorf("no choice given; pass it as first arg: <tool> analytics opt <choice>")
	}
	choiceStr := args[0]
	err := SetOpt(choiceStr)
	if err != nil {
		return err
	}
	d, err := dirs.UseWindmillDir()
	if err != nil {
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
