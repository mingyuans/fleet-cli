package executor

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xq-yan/fleet-cli/internal/manifest"
	"github.com/xq-yan/fleet-cli/internal/output"
)

// ResultStatus represents the outcome of a project operation.
type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusSkip    ResultStatus = "skip"
	StatusFail    ResultStatus = "fail"
)

// Result holds the outcome of a single project operation.
type Result struct {
	Project manifest.ResolvedProject
	Status  ResultStatus
	Label   string // e.g. "cloned", "configured", "skipped", "fetched", "rebased"
	Message string // error or info message
}

// LogFunc is a thread-safe function for printing progress messages within a project operation.
type LogFunc func(format string, args ...any)

// ProjectFunc is the function executed for each project.
type ProjectFunc func(proj manifest.ResolvedProject, log LogFunc) (label string, status ResultStatus, message string)

// Run executes fn for each project in parallel with concurrency limit.
// Results are printed in real-time as each project completes.
func Run(projects []manifest.ResolvedProject, concurrency int, fn ProjectFunc) []Result {
	if concurrency < 1 {
		concurrency = 4
	}

	total := len(projects)
	results := make([]Result, total)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var printMu sync.Mutex
	var done atomic.Int32

	for i, proj := range projects {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, p manifest.ResolvedProject) {
			defer wg.Done()
			defer func() { <-sem }()

			id := p.Path
			log := func(format string, args ...any) {
				printMu.Lock()
				output.PendingSet(id, fmt.Sprintf("%s: %s", p.Path, fmt.Sprintf(format, args...)))
				printMu.Unlock()
			}

			label, status, message := fn(p, log)
			results[idx] = Result{
				Project: p,
				Status:  status,
				Label:   label,
				Message: message,
			}

			completed := int(done.Add(1))
			printMu.Lock()
			output.PendingRemove(id)
			output.PendingFlush(func() {
				prefix := output.Progress(completed, total)
				printResult(prefix, results[idx])
			})
			printMu.Unlock()
		}(i, proj)
	}
	wg.Wait()

	return results
}

func printResult(prefix string, r Result) {
	switch r.Status {
	case StatusSuccess:
		output.Success("%s %s (%s)", prefix, r.Project.Path, r.Label)
	case StatusSkip:
		if r.Message != "" {
			output.Skip("%s %s (%s: %s)", prefix, r.Project.Path, r.Label, r.Message)
		} else {
			output.Skip("%s %s (%s)", prefix, r.Project.Path, r.Label)
		}
	case StatusFail:
		output.Error("%s %s: %s", prefix, r.Project.Path, r.Message)
	}
}

// CountResults counts results by label.
// Labels may contain details after a space (e.g. "created from github/master");
// only the first word is used as the summary key.
func CountResults(results []Result) output.SummaryCounts {
	counts := make(output.SummaryCounts)
	for _, r := range results {
		key, _, _ := strings.Cut(r.Label, " ")
		counts[key]++
	}
	return counts
}
