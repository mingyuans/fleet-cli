package executor

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xq-yan/fleet-cli/internal/manifest"
)

func makeProjects(n int) []manifest.ResolvedProject {
	projects := make([]manifest.ResolvedProject, n)
	for i := range n {
		projects[i] = manifest.ResolvedProject{
			Name: fmt.Sprintf("proj-%d", i),
			Path: fmt.Sprintf("services/proj-%d", i),
		}
	}
	return projects
}

func TestRunConcurrencyLimit(t *testing.T) {
	projects := makeProjects(10)
	var maxConcurrent atomic.Int32
	var current atomic.Int32

	results := Run(projects, 3, func(proj manifest.ResolvedProject, buf *bytes.Buffer, log LogFunc) (string, ResultStatus, string) {
		c := current.Add(1)
		for {
			prev := maxConcurrent.Load()
			if c <= prev || maxConcurrent.CompareAndSwap(prev, c) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		current.Add(-1)
		return "done", StatusSuccess, ""
	})

	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}
	if maxConcurrent.Load() > 3 {
		t.Errorf("expected max concurrency 3, got %d", maxConcurrent.Load())
	}
}

func TestRunResultOrdering(t *testing.T) {
	projects := makeProjects(5)

	results := Run(projects, 2, func(proj manifest.ResolvedProject, buf *bytes.Buffer, log LogFunc) (string, ResultStatus, string) {
		return proj.Name, StatusSuccess, ""
	})

	for i, r := range results {
		expected := fmt.Sprintf("proj-%d", i)
		if r.Label != expected {
			t.Errorf("result[%d]: expected label %q, got %q", i, expected, r.Label)
		}
	}
}

func TestRunMixedResults(t *testing.T) {
	projects := makeProjects(3)

	results := Run(projects, 4, func(proj manifest.ResolvedProject, buf *bytes.Buffer, log LogFunc) (string, ResultStatus, string) {
		switch proj.Name {
		case "proj-0":
			return "cloned", StatusSuccess, ""
		case "proj-1":
			return "skipped", StatusSkip, ""
		default:
			return "failed", StatusFail, "error occurred"
		}
	})

	counts := CountResults(results)
	if counts["cloned"] != 1 || counts["skipped"] != 1 || counts["failed"] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}
