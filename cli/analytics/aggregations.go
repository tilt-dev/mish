package analytics

type Aggregation func([]Event) (agg interface{}, anon interface{}, err error)

func Nil([]Event) (interface{}, interface{}, error) {
	return nil, nil, nil
}

func SaveLocallyReportUnregistered(es []Event) (interface{}, interface{}, error) {
	return es, "Unregistered", nil
}
