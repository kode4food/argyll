package engine

import (
	"log/slog"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// stepOutcome is how a measured Step finished
type stepOutcome int

const (
	metricStep  = "step"
	metricValue = "value"
)

const (
	outcomeSucceeded stepOutcome = iota
	outcomeFailed
	outcomeSkipped
)

// stepMetrics selects each metric Attribute's value from a completed, failed, or skipped Step's
// measured work, tagged with the Step that produced it. A failed select omits its metric
func (tx *flowTx) stepMetrics(st *api.Step, outcome stepOutcome) api.Args {
	var metrics api.Args
	res := api.Args{}
	for name, attr := range st.Attributes {
		if !attr.IsMetric() {
			continue
		}
		if metrics == nil {
			ex := tx.Value().Executions[st.ID]
			metrics = tx.measureMetrics(&ex, outcome)
		}
		value, err := tx.scripts.Evaluate(script.MetricStep, attr.MetricSelect(), metrics)
		if err != nil {
			slog.Warn("Metric select failed, omitting metric",
				slog.String("attribute", string(name)),
				log.StepID(st.ID),
				log.Error(err))
			continue
		}
		res[name] = map[string]any{
			metricStep:  string(st.ID),
			metricValue: value,
		}
	}
	return res
}

// measureMetrics builds every metric of a finished Step. A skipped Step never started, so it
// measures no elapsed time
func (tx *flowTx) measureMetrics(ex *api.ExecutionState, outcome stepOutcome) api.Args {
	elapsed := 0.0
	if outcome != outcomeSkipped {
		elapsed = float64(max(tx.Now().Sub(ex.StartedAt).Milliseconds(), 0))
	}
	return api.Args{
		api.MetricSucceeded: outcome == outcomeSucceeded,
		api.MetricSkipped:   outcome == outcomeSkipped,
		api.MetricElapsed:   elapsed,
		api.MetricWork:      workMetric(ex.WorkItems),
	}
}

func workMetric(items api.WorkItems) map[string]any {
	failed := 0
	for _, work := range items {
		if work.Status == api.WorkFailed {
			failed++
		}
	}
	return map[string]any{
		api.MetricItems:    float64(len(items)),
		api.MetricFailed:   float64(failed),
		api.MetricAttempts: attemptsMetric(items),
	}
}

// attemptsMetric counts every invocation across a Step's work items, retries included
func attemptsMetric(items api.WorkItems) map[string]any {
	total := 0
	lo := 0
	hi := 0
	for _, work := range items {
		n := work.RetryCount + 1
		total += n
		if lo == 0 || n < lo {
			lo = n
		}
		hi = max(hi, n)
	}
	avg := 0.0
	if len(items) > 0 {
		avg = float64(total) / float64(len(items))
	}
	return map[string]any{
		api.MetricTotal:   float64(total),
		api.MetricAverage: avg,
		api.MetricMin:     float64(lo),
		api.MetricMax:     float64(hi),
	}
}
