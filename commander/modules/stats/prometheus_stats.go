package stats

import (
	"time"

	"golang.org/x/net/context"

	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
)

const (
	DefaultStepDuration = "15s"
)

type Sample struct {
	Time  time.Time
	Value float64
}

func PrometheusStats(name string, duration time.Duration) ([]Sample, error) {
	pClient, err := prometheus.New(prometheus.Config{Address: "http://172.17.0.47:9090"})
	if err != nil {
		return nil, err
	}

	step, _ := time.ParseDuration(DefaultStepDuration)

	pRange := prometheus.Range{
		Start: time.Now().Add(-duration),
		End:   time.Now(),
		Step:  step,
	}

	values, err := prometheus.
		NewQueryAPI(pClient).
		QueryRange(context.TODO(), "node_cpu{cpu=\"cpu0\"}", pRange)

	if err != nil {
		return nil, err
	}

	matrix, ok := values.(model.Matrix)
	if !ok {
		return nil, err
	}

	ret := []Sample{}
	for _, ss := range matrix {
		for _, s := range ss.Values {
			ret = append(ret, Sample{
				Time:  s.Timestamp.Time(),
				Value: float64(s.Value),
			})
		}
	}

	return ret, nil
}
