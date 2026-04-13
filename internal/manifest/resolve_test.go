package manifest

import (
	"testing"
)

func TestResolveInheritsDefaults(t *testing.T) {
	m := &Manifest{
		Remotes: []Remote{
			{Name: "github", Fetch: "git@github.com:Org/"},
			{Name: "fork", Fetch: "git@github.com:User/"},
		},
		Default:  &Default{Remote: "github", Revision: "master", Push: "fork"},
		Projects: []Project{{Name: "my-service", Path: "services/my-service"}},
	}
	resolved, _, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved project, got %d", len(resolved))
	}
	rp := resolved[0]
	if rp.Remote != "github" {
		t.Errorf("expected remote=github, got %q", rp.Remote)
	}
	if rp.Revision != "master" {
		t.Errorf("expected revision=master, got %q", rp.Revision)
	}
	if rp.Push != "fork" {
		t.Errorf("expected push=fork, got %q", rp.Push)
	}
	if rp.CloneURL != "git@github.com:Org/my-service.git" {
		t.Errorf("unexpected clone URL: %q", rp.CloneURL)
	}
	if rp.PushURL != "git@github.com:User/my-service.git" {
		t.Errorf("unexpected push URL: %q", rp.PushURL)
	}
	if !rp.HasPushRemote {
		t.Error("expected HasPushRemote=true")
	}
}

func TestResolveProjectOverrides(t *testing.T) {
	m := &Manifest{
		Remotes: []Remote{
			{Name: "github", Fetch: "git@github.com:Org/"},
			{Name: "custom", Fetch: "git@custom.com:Team/"},
		},
		Default:  &Default{Remote: "github", Revision: "master"},
		Projects: []Project{{Name: "svc", Path: "services/svc", Remote: "custom"}},
	}
	resolved, _, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rp := resolved[0]
	if rp.Remote != "custom" {
		t.Errorf("expected remote=custom, got %q", rp.Remote)
	}
	if rp.CloneURL != "git@custom.com:Team/svc.git" {
		t.Errorf("unexpected clone URL: %q", rp.CloneURL)
	}
}

func TestResolveNoPushRemote(t *testing.T) {
	m := &Manifest{
		Remotes:  []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
		Default:  &Default{Remote: "github", Revision: "master"},
		Projects: []Project{{Name: "svc", Path: "services/svc"}},
	}
	resolved, _, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rp := resolved[0]
	if rp.HasPushRemote {
		t.Error("expected HasPushRemote=false when no push remote")
	}
	if rp.PushURL != "" {
		t.Errorf("expected empty push URL, got %q", rp.PushURL)
	}
}

func TestResolveSyncJ(t *testing.T) {
	m := &Manifest{
		Remotes:  []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
		Default:  &Default{Remote: "github", Revision: "master", SyncJ: "8"},
		Projects: []Project{{Name: "svc", Path: "services/svc"}},
	}
	_, syncJ, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if syncJ != 8 {
		t.Errorf("expected syncJ=8, got %d", syncJ)
	}
}

func TestResolveSyncJDefault(t *testing.T) {
	m := &Manifest{
		Remotes:  []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
		Default:  &Default{Remote: "github", Revision: "master"},
		Projects: []Project{{Name: "svc", Path: "services/svc"}},
	}
	_, syncJ, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if syncJ != 4 {
		t.Errorf("expected syncJ=4 (default), got %d", syncJ)
	}
}

func TestResolveGroups(t *testing.T) {
	m := &Manifest{
		Remotes:  []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
		Default:  &Default{Remote: "github", Revision: "master"},
		Projects: []Project{{Name: "svc", Path: "services/svc", Groups: "core,backend"}},
	}
	resolved, _, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rp := resolved[0]
	if len(rp.Groups) != 2 || rp.Groups[0] != "core" || rp.Groups[1] != "backend" {
		t.Errorf("unexpected groups: %v", rp.Groups)
	}
}

func TestResolveUnknownRemoteError(t *testing.T) {
	m := &Manifest{
		Remotes:  []Remote{{Name: "github", Fetch: "git@github.com:Org/"}},
		Default:  &Default{Remote: "unknown", Revision: "master"},
		Projects: []Project{{Name: "svc", Path: "services/svc"}},
	}
	_, _, _, err := Resolve(m)
	if err == nil {
		t.Fatal("expected error for unknown remote")
	}
}
