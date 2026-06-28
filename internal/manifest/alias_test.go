package manifest

import (
	"encoding/xml"
	"reflect"
	"testing"
)

func TestNormalizeAliasGroups(t *testing.T) {
	tests := []struct {
		name             string
		aliases          []BranchAlias
		masterMainCompat bool
		want             [][]string
	}{
		{
			name:    "single group",
			aliases: []BranchAlias{{Branches: []string{"testing", "testing-incy"}}},
			want:    [][]string{{"testing", "testing-incy"}},
		},
		{
			name:    "trim and drop blank members",
			aliases: []BranchAlias{{Branches: []string{" testing-incy ", "  ", "testing"}}},
			want:    [][]string{{"testing-incy", "testing"}},
		},
		{
			name:    "drop intra-group duplicates",
			aliases: []BranchAlias{{Branches: []string{"testing", "testing", "testing-incy"}}},
			want:    [][]string{{"testing", "testing-incy"}},
		},
		{
			name:    "ignore single-member group",
			aliases: []BranchAlias{{Branches: []string{"testing"}}},
			want:    nil,
		},
		{
			name: "multiple groups",
			aliases: []BranchAlias{
				{Branches: []string{"testing", "testing-incy"}},
				{Branches: []string{"staging", "staging-incy"}},
			},
			want: [][]string{{"testing", "testing-incy"}, {"staging", "staging-incy"}},
		},
		{
			name:             "master-main-compat injects builtin group",
			aliases:          nil,
			masterMainCompat: true,
			want:             [][]string{{"master", "main"}},
		},
		{
			name:             "master-main-compat not injected when explicit group covers master",
			aliases:          []BranchAlias{{Branches: []string{"master", "main", "trunk"}}},
			masterMainCompat: true,
			want:             [][]string{{"master", "main", "trunk"}},
		},
		{
			name:             "no master-main fallback when disabled",
			aliases:          nil,
			masterMainCompat: false,
			want:             nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAliasGroups(tt.aliases, tt.masterMainCompat)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizeAliasGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveInjectsAliasGroups(t *testing.T) {
	m := &Manifest{
		Remotes: []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
		Default: &Default{Remote: "github", Revision: "master", MasterMainCompat: "true"},
		BranchAliases: []BranchAlias{
			{Branches: []string{"testing", "testing-incy"}},
		},
		Projects: []Project{{Name: "svc", Path: "services/svc"}},
	}
	resolved, _, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]string{{"testing", "testing-incy"}, {"master", "main"}}
	if !reflect.DeepEqual(resolved[0].AliasGroups, want) {
		t.Errorf("AliasGroups = %v, want %v", resolved[0].AliasGroups, want)
	}
}

func TestParseBranchAliasXML(t *testing.T) {
	data := []byte(`<manifest>
  <branch-alias>
    <branch>testing</branch>
    <branch>testing-incy</branch>
  </branch-alias>
  <branch-alias>
    <branch>staging</branch>
    <branch>staging-incy</branch>
  </branch-alias>
</manifest>`)

	var m Manifest
	if err := xml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m.BranchAliases) != 2 {
		t.Fatalf("expected 2 branch-alias groups, got %d", len(m.BranchAliases))
	}
	if !reflect.DeepEqual(m.BranchAliases[0].Branches, []string{"testing", "testing-incy"}) {
		t.Errorf("group 0 = %v", m.BranchAliases[0].Branches)
	}
	if !reflect.DeepEqual(m.BranchAliases[1].Branches, []string{"staging", "staging-incy"}) {
		t.Errorf("group 1 = %v", m.BranchAliases[1].Branches)
	}
}

func TestMergeBranchAliases(t *testing.T) {
	base := &Manifest{
		BranchAliases: []BranchAlias{{Branches: []string{"master", "main"}}},
	}
	local := &Manifest{
		BranchAliases: []BranchAlias{{Branches: []string{"testing", "testing-incy"}}},
	}
	merged := Merge(base, local)
	if len(merged.BranchAliases) != 2 {
		t.Fatalf("expected 2 alias groups after merge, got %d", len(merged.BranchAliases))
	}
	if !reflect.DeepEqual(merged.BranchAliases[0].Branches, []string{"master", "main"}) {
		t.Errorf("group 0 = %v", merged.BranchAliases[0].Branches)
	}
	if !reflect.DeepEqual(merged.BranchAliases[1].Branches, []string{"testing", "testing-incy"}) {
		t.Errorf("group 1 = %v", merged.BranchAliases[1].Branches)
	}
}

func TestMergeBranchAliasesReplacesSameMemberSet(t *testing.T) {
	// Same member set (order-independent) should be replaced, not duplicated.
	base := &Manifest{
		BranchAliases: []BranchAlias{{Branches: []string{"testing", "testing-incy"}}},
	}
	local := &Manifest{
		BranchAliases: []BranchAlias{{Branches: []string{"testing-incy", "testing"}}},
	}
	merged := Merge(base, local)
	if len(merged.BranchAliases) != 1 {
		t.Fatalf("expected 1 alias group after merge (same member set), got %d", len(merged.BranchAliases))
	}
	// The local definition wins.
	if !reflect.DeepEqual(merged.BranchAliases[0].Branches, []string{"testing-incy", "testing"}) {
		t.Errorf("merged group = %v", merged.BranchAliases[0].Branches)
	}
}
