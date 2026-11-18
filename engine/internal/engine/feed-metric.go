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
// measured work, tagged with the Step that produced it. A failed selector omits its metric
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
		value, err := tx.scripts.Evaluate(script.MetricStep, attr.MetricSelector(), metrics)
		if err != nil {
			slog.Warn("Metric selector failed, omitting metric",
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
// measures no elapsed time. A Step measured after it settled ends at its completion
func (tx *flowTx) measureMetrics(ex *api.ExecutionState, outcome stepOutcome) api.Args {
	end := tx.Now()
	if !ex.Completion.At.IsZero() {
		end = ex.Completion.At
	}
	elapsed := 0.0
	if outcome != outcomeSkipped {
		elapsed = float64(max(end.Sub(ex.StartedAt).Milliseconds(), 0))
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
	res := map[string]any{
		api.MetricItems:    float64(len(items)),
		api.MetricFailed:   float64(failed),
		api.MetricAttempts: attemptsMetric(items),
	}
	if usage := workUsage(items); usage != nil {
		res[api.MetricUsage] = usage
	}
	return res
}

// workUsage is the usage metric of a Step's work items, nil when none reported any
func workUsage(items api.WorkItems) map[string]any {
	sum := sumUsage(items)
	if sum == nil {
		return nil
	}
	res := make(map[string]any, len(sum))
	for k, v := range sum {
		res[k] = v
	}
	return res
}

// sumUsage sums the usage work items reported, nil when none reported any
func sumUsage(items api.WorkItems) api.Usage {
	var res api.Usage
	for _, work := range items {
		res = res.Add(work.Usage)
	}
	return res
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
