package manifest

import (
	"reflect"
	"testing"
)

func TestMergeRemotesReplace(t *testing.T) {
	base := &Manifest{
		Remotes: []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
	}
	local := &Manifest{
		Remotes: []Remote{{Name: "github", Fetch: "git@github.com:User/"}},
	}
	result := Merge(base, local)
	if len(result.Remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(result.Remotes))
	}
	if result.Remotes[0].Fetch != "git@github.com:User/" {
		t.Errorf("expected replaced fetch, got %q", result.Remotes[0].Fetch)
	}
}

func TestMergeRemotesAppend(t *testing.T) {
	base := &Manifest{
		Remotes: []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
	}
	local := &Manifest{
		Remotes: []Remote{{Name: "fork", Fetch: "git@github.com:user/"}},
	}
	result := Merge(base, local)
	if len(result.Remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(result.Remotes))
	}
}

func TestMergeDefaultPerAttribute(t *testing.T) {
	base := &Manifest{
		Default: &Default{Remote: "github", Revision: "master", SyncJ: "4"},
	}
	local := &Manifest{
		Default: &Default{Push: "fork"},
	}
	result := Merge(base, local)
	d := result.Default
	if d.Remote != "github" || d.Revision != "master" || d.SyncJ != "4" || d.Push != "fork" {
		t.Errorf("unexpected merged default: %+v", d)
	}
}

func TestMergeDefaultMultipleOverrides(t *testing.T) {
	base := &Manifest{
		Default: &Default{Remote: "github", Revision: "master"},
	}
	local := &Manifest{
		Default: &Default{Revision: "main", SyncJ: "8"},
	}
	result := Merge(base, local)
	d := result.Default
	if d.Remote != "github" || d.Revision != "main" || d.SyncJ != "8" {
		t.Errorf("unexpected merged default: %+v", d)
	}
}

func TestMergeDefaultNilBase(t *testing.T) {
	base := &Manifest{}
	local := &Manifest{
		Default: &Default{Push: "fork"},
	}
	result := Merge(base, local)
	if result.Default == nil || result.Default.Push != "fork" {
		t.Errorf("expected default with push=fork, got %+v", result.Default)
	}
}

func TestMergeDefaultNilLocal(t *testing.T) {
	base := &Manifest{
		Default: &Default{Remote: "github", Revision: "master"},
	}
	local := &Manifest{}
	result := Merge(base, local)
	if result.Default.Remote != "github" || result.Default.Revision != "master" {
		t.Errorf("expected base default preserved, got %+v", result.Default)
	}
}

func TestMergeProjectPerAttribute(t *testing.T) {
	base := &Manifest{
		Projects: []Project{{Name: "svc", Path: "services/svc", Revision: "master"}},
	}
	local := &Manifest{
		Projects: []Project{{Name: "svc", Revision: "develop"}},
	}
	result := Merge(base, local)
	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(result.Projects))
	}
	p := result.Projects[0]
	if p.Path != "services/svc" || p.Revision != "develop" {
		t.Errorf("unexpected merged project: %+v", p)
	}
}

func TestMergeProjectAppend(t *testing.T) {
	base := &Manifest{
		Projects: []Project{{Name: "svc-a", Path: "services/svc-a"}},
	}
	local := &Manifest{
		Projects: []Project{{Name: "svc-b", Path: "services/svc-b"}},
	}
	result := Merge(base, local)
	if len(result.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(result.Projects))
	}
}

func TestMergeDefaultWorktreeBaseFromLocalOnly(t *testing.T) {
	base := &Manifest{}
	local := &Manifest{
		Default: &Default{WorktreeBase: "./worktrees/fleet"},
	}
	result := Merge(base, local)
	if result.Default == nil || result.Default.WorktreeBase != "./worktrees/fleet" {
		t.Errorf("expected worktree-base from local, got %+v", result.Default)
	}
}

func TestMergeDefaultWorktreeBaseOverride(t *testing.T) {
	base := &Manifest{
		Default: &Default{Remote: "github", WorktreeBase: "~/worktrees/team"},
	}
	local := &Manifest{
		Default: &Default{WorktreeBase: "~/worktrees/mine"},
	}
	result := Merge(base, local)
	if result.Default.WorktreeBase != "~/worktrees/mine" {
		t.Errorf("expected local worktree-base to win, got %q", result.Default.WorktreeBase)
	}
	if result.Default.Remote != "github" {
		t.Errorf("expected base remote preserved, got %q", result.Default.Remote)
	}
}

func TestMergeDefaultWorktreeBasePreserved(t *testing.T) {
	base := &Manifest{
		Default: &Default{WorktreeBase: "~/worktrees/team"},
	}
	local := &Manifest{
		Default: &Default{Revision: "main"},
	}
	result := Merge(base, local)
	if result.Default.WorktreeBase != "~/worktrees/team" {
		t.Errorf("expected base worktree-base preserved, got %q", result.Default.WorktreeBase)
	}
}

func TestMergeDefaultWorktreeCopy(t *testing.T) {
	base := &Manifest{
		Default: &Default{WorktreeCopy: ".env"},
	}
	local := &Manifest{
		Default: &Default{WorktreeCopy: ".env,.env.*,config.local.yaml"},
	}
	result := Merge(base, local)
	if result.Default.WorktreeCopy != ".env,.env.*,config.local.yaml" {
		t.Errorf("expected local worktree-copy to win, got %q", result.Default.WorktreeCopy)
	}
}

func TestMergeDefaultWorktreeCopyPreserved(t *testing.T) {
	base := &Manifest{
		Default: &Default{WorktreeCopy: ".env"},
	}
	local := &Manifest{
		Default: &Default{Push: "fork"},
	}
	result := Merge(base, local)
	if result.Default.WorktreeCopy != ".env" {
		t.Errorf("expected base worktree-copy preserved, got %q", result.Default.WorktreeCopy)
	}
}

func TestMergeProjectWorktreeCopyOverride(t *testing.T) {
	base := &Manifest{
		Projects: []Project{{Name: "api.git", Path: "services/api", WorktreeCopy: ".env"}},
	}
	local := &Manifest{
		Projects: []Project{{Name: "api.git", WorktreeCopy: ".env,secrets.local"}},
	}
	result := Merge(base, local)
	p := result.Projects[0]
	if p.WorktreeCopy != ".env,secrets.local" {
		t.Errorf("expected local worktree-copy to win, got %q", p.WorktreeCopy)
	}
	if p.Path != "services/api" {
		t.Errorf("expected base path preserved, got %q", p.Path)
	}
}

func TestMergeProjectWorktreeCopyPreserved(t *testing.T) {
	base := &Manifest{
		Projects: []Project{{Name: "api.git", Path: "services/api", WorktreeCopy: ".env"}},
	}
	local := &Manifest{
		Projects: []Project{{Name: "api.git", Revision: "develop"}},
	}
	result := Merge(base, local)
	p := result.Projects[0]
	if p.WorktreeCopy != ".env" {
		t.Errorf("expected base worktree-copy preserved, got %q", p.WorktreeCopy)
	}
	if p.Revision != "develop" {
		t.Errorf("expected local revision applied, got %q", p.Revision)
	}
}

// TestMergeDefaultEveryAttributeOverridable guards against adding a new <default>
// attribute to the struct without a matching branch in mergeDefault. Each case
// sets exactly one attribute in the local manifest and asserts it survives the
// merge; a field left out of mergeDefault fails here instead of silently
// vanishing from local_fleet.xml at runtime.
func TestMergeDefaultEveryAttributeOverridable(t *testing.T) {
	cases := []struct {
		attr  string
		local Default
		get   func(*Default) string
	}{
		{"remote", Default{Remote: "local-remote"}, func(d *Default) string { return d.Remote }},
		{"revision", Default{Revision: "local-revision"}, func(d *Default) string { return d.Revision }},
		{"sync-j", Default{SyncJ: "9"}, func(d *Default) string { return d.SyncJ }},
		{"push", Default{Push: "local-push"}, func(d *Default) string { return d.Push }},
		{"master-main-compat", Default{MasterMainCompat: "true"}, func(d *Default) string { return d.MasterMainCompat }},
		{"worktree-base", Default{WorktreeBase: "local-base"}, func(d *Default) string { return d.WorktreeBase }},
		{"worktree-copy", Default{WorktreeCopy: "local-copy"}, func(d *Default) string { return d.WorktreeCopy }},
	}

	if got := reflect.TypeOf(Default{}).NumField(); got != len(cases) {
		t.Fatalf("Default has %d fields but only %d are covered; add the new attribute to mergeDefault and to this table", got, len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.attr, func(t *testing.T) {
			base := &Manifest{Default: &Default{}}
			local := tc.local
			result := Merge(base, &Manifest{Default: &local})
			want := tc.get(&local)
			if got := tc.get(result.Default); got != want {
				t.Errorf("attribute %s not merged: got %q, want %q", tc.attr, got, want)
			}
		})
	}
}

// TestMergeProjectEveryAttributeOverridable is the <project> counterpart to
// TestMergeDefaultEveryAttributeOverridable. Name is excluded because it is the
// match key, not an overridable attribute.
func TestMergeProjectEveryAttributeOverridable(t *testing.T) {
	cases := []struct {
		attr  string
		local Project
		get   func(*Project) string
	}{
		{"path", Project{Path: "local/path"}, func(p *Project) string { return p.Path }},
		{"groups", Project{Groups: "local-group"}, func(p *Project) string { return p.Groups }},
		{"remote", Project{Remote: "local-remote"}, func(p *Project) string { return p.Remote }},
		{"revision", Project{Revision: "local-revision"}, func(p *Project) string { return p.Revision }},
		{"push", Project{Push: "local-push"}, func(p *Project) string { return p.Push }},
		{"worktree-copy", Project{WorktreeCopy: "local-copy"}, func(p *Project) string { return p.WorktreeCopy }},
	}

	const nameField = 1
	if got := reflect.TypeOf(Project{}).NumField(); got != len(cases)+nameField {
		t.Fatalf("Project has %d fields but only %d overridable ones are covered; add the new attribute to mergeProjects and to this table", got, len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.attr, func(t *testing.T) {
			base := &Manifest{Projects: []Project{{Name: "svc"}}}
			local := tc.local
			local.Name = "svc"
			result := Merge(base, &Manifest{Projects: []Project{local}})
			if len(result.Projects) != 1 {
				t.Fatalf("expected 1 project, got %d", len(result.Projects))
			}
			want := tc.get(&local)
			if got := tc.get(&result.Projects[0]); got != want {
				t.Errorf("attribute %s not merged: got %q, want %q", tc.attr, got, want)
			}
		})
	}
}
