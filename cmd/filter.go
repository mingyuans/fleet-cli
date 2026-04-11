package cmd

import (
	"slices"
	"strings"

	"github.com/xq-yan/fleet-cli/internal/manifest"
)

// filterByGroup filters projects by the global group filter.
// Supports "," for OR and "+" for AND:
//   - "core,web"       → projects in core OR web
//   - "core+backend"   → projects in core AND backend
//   - "core+backend,infra" → (core AND backend) OR infra
func filterByGroup(projects []manifest.ResolvedProject) []manifest.ResolvedProject {
	if groupFilter == "" {
		return projects
	}
	orTerms := strings.Split(groupFilter, ",")
	var filtered []manifest.ResolvedProject
	for _, p := range projects {
		if matchGroupExpr(p.Groups, orTerms) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// matchGroupExpr checks if a project's groups satisfy any of the OR terms.
// Each OR term may contain "+" separated AND conditions.
func matchGroupExpr(groups []string, orTerms []string) bool {
	for _, term := range orTerms {
		andParts := strings.Split(term, "+")
		if matchAll(groups, andParts) {
			return true
		}
	}
	return false
}

// matchAll returns true if groups contains every value in required.
func matchAll(groups []string, required []string) bool {
	for _, r := range required {
		if !slices.Contains(groups, r) {
			return false
		}
	}
	return true
}
