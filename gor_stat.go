package goreplay

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"lf.git.oa.mt/go-component/metrics"
)

type GorStat struct {
	statName string
	address  string
	rateMs   int
	latest   int
	mean     int
	max      int
	count    int
}

func NewGorStat(statName string, address string, rateMs int) (s *GorStat) {
	s = new(GorStat)
	s.statName = statName
	s.address = address
	s.rateMs = rateMs
	s.latest = 0
	s.mean = 0
	s.max = 0
	s.count = 0

	if Settings.Stats {
		go s.reportStats()
	}
	return
}

func (s *GorStat) Write(latest int) {
	if Settings.Stats {
		if latest > s.max {
			s.max = latest
		}
		if latest != 0 {
			s.mean = ((s.mean * s.count) + latest) / (s.count + 1)
		}
		s.latest = latest
		s.count = s.count + 1
	}
}

func (s *GorStat) Reset() {
	s.latest = 0
	s.max = 0
	s.mean = 0
	s.count = 0
}

func (s *GorStat) String() string {
	return s.statName + ":" + s.address + "," + strconv.Itoa(s.latest) + "," + strconv.Itoa(s.mean) + "," + strconv.Itoa(s.max) + "," + strconv.Itoa(s.count) + "," + strconv.Itoa(s.count/(s.rateMs/1000.0)) + "," + strconv.Itoa(runtime.NumGoroutine())
}

func (s *GorStat) reportStats() {
	Debug(0, "\n", s.statName+":address,latest,mean,max,count,count/second,gcount")
	for {
		Debug(0, "\n", s)
		err := s.prome()
		if err != nil {
			Debug(0, "GorStat prome error:", err.Error())
		}
		s.Reset()
		time.Sleep(time.Duration(s.rateMs) * time.Millisecond)
	}
}

// report stat to prometheus
func (s *GorStat) prome() (err error) {
	defer func() {
		if r := recover(); r != nil {
			Debug(0, "GorStat prome:", string(debug.Stack()))
		}
	}()
	// disable prometheus
	if Settings.PrometheusDisabled {
		return
	}
	// 指标上报
	err = metrics.GaugeSet(Metrics.OutputHttpStat, float64(s.latest), map[string]string{
		"address": s.address,
		"stat":    "latest",
	})
	if err != nil {
		return
	}
	err = metrics.GaugeSet(Metrics.OutputHttpStat, float64(s.mean), map[string]string{
		"address": s.address,
		"stat":    "mean",
	})
	if err != nil {
		return
	}
	err = metrics.GaugeSet(Metrics.OutputHttpStat, float64(s.max), map[string]string{
		"address": s.address,
		"stat":    "max",
	})
	if err != nil {
		return
	}
	err = metrics.GaugeSet(Metrics.OutputHttpStat, float64(s.count), map[string]string{
		"address": s.address,
		"stat":    "count",
	})
	if err != nil {
		return
	}
	err = metrics.GaugeSet(Metrics.OutputHttpStat, float64(s.count/(s.rateMs/1000.0)), map[string]string{
		"address": s.address,
		"stat":    "count/second",
	})
	if err != nil {
		return
	}
	return
}
