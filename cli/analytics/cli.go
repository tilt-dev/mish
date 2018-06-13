package analytics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/windmilleng/mish/cli/dirs"
)

func analyticsStatus(_ *cobra.Command, args []string) error {
	// check for the choice.txt file

	fmt.Println("[ for more info: https://windmill.engineering/analytics ]")

	d, err := dirs.UseWindmillDir()
	if err != nil {
		return err
	}

	txt, err := d.ReadFile(choiceFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		txt = ""
	}

	txt = strings.TrimSpace(txt)

	choice := CollectDefault

	switch txt {
	case choices[CollectNotEvenLocal]:
		choice = CollectNotEvenLocal
	case choices[CollectLocal]:
		choice = CollectLocal
	case choices[CollectUpload]:
		choice = CollectUpload
	}

	fmt.Printf("current collection strategy: %v\n", choices[choice])

	return nil
}

func analyticsOpt(_ *cobra.Command, args []string) (outerErr error) {
	fmt.Printf("choice can be one of {%v, %v, %v}\n", choices[CollectUpload], choices[CollectLocal], choices[CollectNotEvenLocal])
	if len(args) == 0 {
		return fmt.Errorf("no choice given; pass it as first arg: <tool> analytics opt <choice>")
	}
	choiceStr := args[0]
	var choice UserCollection
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

	fmt.Printf("wrote user collection strategy %q to file %v", choiceStr, filepath.Join(d.Root(), choiceFile))
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
