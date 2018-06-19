package mish

import (
	"github.com/windmilleng/mish/cli/analytics"
)

var mishlytics analytics.Analytics

func initAnalytics() (analytics.Analytics, error) {
	a, _, err := analytics.Init()
	if err != nil {
		return nil, err
	}

	mishlytics = a

	return a, nil
}
