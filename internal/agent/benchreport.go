package agent

import (
	"encoding/json"
	"time"

	"github.com/FacileStudio/nacelle"
)

// benchTotals sums the successful runs' measurements; a failed run contributes
// nothing, since its numbers stopped partway through.
type benchTotals struct {
	runs               int
	duration           time.Duration
	usage              nacelle.Usage
	toolCalls          int
	finalContextTokens int64
}

// newBenchTotals sums every run that finished clean.
func newBenchTotals(results []benchRun) benchTotals {
	var t benchTotals
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		t.runs++
		t.duration += r.Duration
		t.usage = t.usage.Add(r.Stats.Usage)
		t.toolCalls += r.Stats.ToolCalls
		t.finalContextTokens += r.Stats.FinalContextTokens
	}
	return t
}

// average divides the totals by the run count, giving the per-run means the
// summary row prints. An empty total divides nothing and comes back as is.
func (t benchTotals) average() benchTotals {
	if t.runs == 0 {
		return t
	}
	n := int64(t.runs)
	return benchTotals{
		runs:     t.runs,
		duration: t.duration / time.Duration(n),
		usage: nacelle.Usage{
			InputTokens:         t.usage.InputTokens / n,
			OutputTokens:        t.usage.OutputTokens / n,
			CacheReadTokens:     t.usage.CacheReadTokens / n,
			CacheCreationTokens: t.usage.CacheCreationTokens / n,
			Cost:                t.usage.Cost / float64(n),
		},
		toolCalls:          t.toolCalls / t.runs,
		finalContextTokens: t.finalContextTokens / n,
	}
}

// benchJSONRun is one run as the bench --json document prints it.
type benchJSONRun struct {
	Run                 int     `json:"run"`
	Status              string  `json:"status"`
	Error               string  `json:"error,omitempty"`
	DurationMs          int64   `json:"duration_ms"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	Cost                float64 `json:"cost"`
	ToolCalls           int     `json:"tool_calls"`
	FinalContextTokens  int64   `json:"final_context_tokens"`
}

// benchJSONSummary is the total or average record of the bench --json
// document.
type benchJSONSummary struct {
	DurationMs          int64   `json:"duration_ms"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	Cost                float64 `json:"cost"`
	ToolCalls           int     `json:"tool_calls"`
	FinalContextTokens  int64   `json:"final_context_tokens"`
}

// benchReport is the one JSON document bench --json prints: every run, the
// count that failed, and the totals and per-run averages of the clean ones.
type benchReport struct {
	Model   string           `json:"model,omitempty"`
	Runs    []benchJSONRun   `json:"runs"`
	Failed  int              `json:"failed"`
	Total   benchJSONSummary `json:"total"`
	Average benchJSONSummary `json:"average"`
}

// jsonRun converts one run's measurements to its JSON record.
func (r benchRun) jsonRun() benchJSONRun {
	rec := benchJSONRun{
		Run:                 r.Run,
		Status:              benchOK,
		DurationMs:          r.Duration.Milliseconds(),
		InputTokens:         r.Stats.InputTokens,
		OutputTokens:        r.Stats.OutputTokens,
		CacheReadTokens:     r.Stats.CacheReadTokens,
		CacheCreationTokens: r.Stats.CacheCreationTokens,
		Cost:                r.Stats.Cost,
		ToolCalls:           r.Stats.ToolCalls,
		FinalContextTokens:  r.Stats.FinalContextTokens,
	}
	if r.Err != nil {
		rec.Status = benchFailed
		rec.Error = r.Err.Error()
	}
	return rec
}

// jsonSummary converts the totals to the JSON summary record.
func (t benchTotals) jsonSummary() benchJSONSummary {
	return benchJSONSummary{
		DurationMs:          t.duration.Milliseconds(),
		InputTokens:         t.usage.InputTokens,
		OutputTokens:        t.usage.OutputTokens,
		CacheReadTokens:     t.usage.CacheReadTokens,
		CacheCreationTokens: t.usage.CacheCreationTokens,
		Cost:                t.usage.Cost,
		ToolCalls:           t.toolCalls,
		FinalContextTokens:  t.finalContextTokens,
	}
}

// newBenchReport assembles the JSON document from the run results.
func newBenchReport(results []benchRun, model string) benchReport {
	totals := newBenchTotals(results)
	report := benchReport{
		Model:   model,
		Runs:    make([]benchJSONRun, 0, len(results)),
		Total:   totals.jsonSummary(),
		Average: totals.average().jsonSummary(),
	}
	for _, r := range results {
		if r.Err != nil {
			report.Failed++
		}
		report.Runs = append(report.Runs, r.jsonRun())
	}
	return report
}

// benchJSON marshals the report the --json flag prints.
func benchJSON(results []benchRun, model string) ([]byte, error) {
	return json.Marshal(newBenchReport(results, model))
}
