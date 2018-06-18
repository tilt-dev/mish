package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/windmilleng/mish/cli/dirs"
	"github.com/windmilleng/mish/logging"
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

type analyticsCLI struct {
	appName  string
	dir      *dirs.WindmillDir
	store    EventStore
	registry map[string]Aggregation
}

func (c *analyticsCLI) gen(_ *cobra.Command, args []string) error {
	// TODO(dbentley): only process certain events, e.g., the last day
	es, err := c.store.Get(time.Unix(0, 0))
	if err != nil {
		return err
	}

	eventByKey := map[string][]Event{}

	for _, e := range es {
		eventByKey[e.Key] = append(eventByKey[e.Key], e)
	}

	agg := map[string]interface{}{}
	anon := map[string]interface{}{}

	for k, v := range eventByKey {
		f, ok := c.registry[k]
		if !ok {
			f = SaveLocallyReportUnregistered
		}
		if f == nil {
			continue
		}

		var err error
		agg[k], anon[k], err = f(v)
		if err != nil {
			logging.Global().Infof("error aggregating key %v: %v", k, err)
		}
	}

	logging.Global().Infof("aggregated: %v", agg)
	logging.Global().Infof("anonymized: %v", anon)

	date := time.Now().UTC().Format("20060102150405")
	logging.Global().Infof("date string %v", date)

	bs, err := json.Marshal(agg)
	if err != nil {
		return err
	}

	p := filepath.Join("analytics", c.appName, "gen", fmt.Sprintf("agg-%s.json", date))
	if err := c.dir.WriteFile(p, string(bs)); err != nil {
		return err
	}

	bs, err = json.Marshal(anon)
	if err != nil {
		return err
	}

	p = filepath.Join("analytics", c.appName, "gen", fmt.Sprintf("anon-%s.json", date))
	if err := c.dir.WriteFile(p, string(bs)); err != nil {
		return err
	}

	// TODO(dbentley): trim events we won't need to aggregate in the future
	return nil
}

func initCLI(s EventStore, appName string, dir *dirs.WindmillDir, registry map[string]Aggregation) (*cobra.Command, error) {
	cli := &analyticsCLI{
		store:    s,
		registry: registry,
		dir:      dir,
		appName:  appName,
	}
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

	gen := &cobra.Command{
		Use:   "gen",
		Short: "generate windmill analytics",
		RunE:  cli.gen,
	}
	analytics.AddCommand(gen)
	return analytics, nil
}
