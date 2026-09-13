package agent

import (
	"fmt"
	"strconv"
	"strings"
)

// benchColumns are the text table's headers, in cell order.
var benchColumns = []string{"run", "duration", "input", "output", "cache", "cost", "tools", "context", "status"}

// benchTextTable renders one row per run plus the total and average summary
// rows, then one line per failed run naming its error.
func benchTextTable(results []benchRun) string {
	rows := [][]string{benchColumns}
	for _, r := range results {
		rows = append(rows, benchRowString(r))
	}
	totals := newBenchTotals(results)
	rows = append(rows, benchSummaryString("total", totals), benchSummaryString("avg", totals.average()))
	out := padBenchTable(rows)
	if notes := benchFailureNotes(results); notes != "" {
		out += notes + "\n"
	}
	return out
}

// benchRowString converts one run to the cells its table row prints, with the
// cache column summing both cache token kinds.
func benchRowString(r benchRun) []string {
	status := benchOK
	if r.Err != nil {
		status = benchFailed
	}
	return []string{
		strconv.Itoa(r.Run),
		r.Duration.String(),
		strconv.FormatInt(r.Stats.InputTokens, 10),
		strconv.FormatInt(r.Stats.OutputTokens, 10),
		strconv.FormatInt(r.Stats.CacheReadTokens+r.Stats.CacheCreationTokens, 10),
		fmt.Sprintf("$%.4f", r.Stats.Cost),
		strconv.Itoa(r.Stats.ToolCalls),
		strconv.FormatInt(r.Stats.FinalContextTokens, 10),
		status,
	}
}

// benchSummaryString converts the totals, or the averages, to the cells their
// summary row prints; the status cell stays empty.
func benchSummaryString(label string, t benchTotals) []string {
	return []string{
		label,
		t.duration.String(),
		strconv.FormatInt(t.usage.InputTokens, 10),
		strconv.FormatInt(t.usage.OutputTokens, 10),
		strconv.FormatInt(t.usage.CacheReadTokens+t.usage.CacheCreationTokens, 10),
		fmt.Sprintf("$%.4f", t.usage.Cost),
		strconv.Itoa(t.toolCalls),
		strconv.FormatInt(t.finalContextTokens, 10),
		"",
	}
}

// benchFailureNotes lists one line per failed run with its error; a clean
// bench prints none.
func benchFailureNotes(results []benchRun) string {
	notes := make([]string, 0, len(results))
	for _, r := range results {
		if r.Err != nil {
			notes = append(notes, fmt.Sprintf("run %d failed: %v", r.Run, r.Err))
		}
	}
	return strings.Join(notes, "\n")
}

// padBenchTable aligns the rows into columns two spaces apart, right-aligned
// everywhere but the trailing status column, with no trailing padding.
func padBenchTable(rows [][]string) string {
	widths := benchColumnWidths(rows)
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			pad := strings.Repeat(" ", widths[i]-len(cell))
			if i == len(row)-1 {
				cells[i] = cell + pad
			} else {
				cells[i] = pad + cell
			}
		}
		lines = append(lines, strings.TrimRight(strings.Join(cells, "  "), " "))
	}
	return strings.Join(lines, "\n") + "\n"
}

// benchColumnWidths measures the widest cell in each column.
func benchColumnWidths(rows [][]string) []int {
	widths := make([]int, len(benchColumns))
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], len(cell))
		}
	}
	return widths
}
