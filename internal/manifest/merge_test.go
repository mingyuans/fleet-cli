package manifest

import (
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
