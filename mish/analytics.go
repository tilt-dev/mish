package mish

import (
	"github.com/windmilleng/mish/cli/analytics"
	"github.com/windmilleng/mish/logging"
)

var mishlytics analytics.Analytics

func checkOptFlag(opt *bool) {

	optIn := *opt

	var err error
	if optIn {
		err = analytics.SetAnalyticsOpt(analytics.AnalyticsOptIn)
	} else {
		err = analytics.SetAnalyticsOpt(analytics.AnalyticsOptOut)
	}
	if err != nil {
		logging.Global().Infof("error setting analytics opt in status: %s", err.Error())
	}
}

func initAnalytics() (analytics.Analytics, error) {
	a, _, err := analytics.Init()
	if err != nil {
		return nil, err
	}

	mishlytics = a

	return a, nil
}
