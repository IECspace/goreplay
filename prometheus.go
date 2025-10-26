package goreplay

import (
	"lf.git.oa.mt/go-component/metrics"
)

var Metrics *PromeStat

type PromeStat struct {
	OutputHttpStat string
}

func NewPromeStat() error {
	Metrics = new(PromeStat)
	Metrics.OutputHttpStat = "output_http_queue_stat"
	// Create a metrics object for the http output queue
	err := metrics.CreateGauge(&metrics.GaugeOpts{
		Name:   Metrics.OutputHttpStat,
		Help:   "The number of requests currently in the output HTTP queue.",
		Labels: []string{"address", "stat"},
	})
	if err != nil {
		return err
	}
	return nil
}
