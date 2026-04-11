package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntegrationParseAndMerge(t *testing.T) {
	dir := t.TempDir()

	defaultXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:ExampleOrg/" review="https://github.com/ExampleOrg/" />
  <default remote="github" revision="master" sync-j="4" />
  <project name="user-service" path="services/user-service" groups="core,be" />
  <project name="order-service" path="services/order-service" groups="products" />
</manifest>`

	localXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="fork" fetch="git@github.com:testuser/" />
  <default push="fork" />
  <project name="order-service" revision="develop" />
</manifest>`

	if err := os.WriteFile(filepath.Join(dir, "default.xml"), []byte(defaultXML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "local.xml"), []byte(localXML), 0644); err != nil {
		t.Fatal(err)
	}

	base, err := ParseFile(filepath.Join(dir, "default.xml"))
	if err != nil {
		t.Fatalf("parse default: %v", err)
	}
	local, err := ParseFile(filepath.Join(dir, "local.xml"))
	if err != nil {
		t.Fatalf("parse local: %v", err)
	}

	merged := Merge(base, local)

	// Verify remotes
	if len(merged.Remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(merged.Remotes))
	}
	remoteMap := make(map[string]Remote)
	for _, r := range merged.Remotes {
		remoteMap[r.Name] = r
	}
	if _, ok := remoteMap["github"]; !ok {
		t.Error("missing github remote")
	}
	if _, ok := remoteMap["fork"]; !ok {
		t.Error("missing fork remote")
	}

	// Verify default merge
	if merged.Default.Push != "fork" {
		t.Errorf("expected push=fork, got %q", merged.Default.Push)
	}
	if merged.Default.Remote != "github" {
		t.Errorf("expected remote=github preserved, got %q", merged.Default.Remote)
	}

	// Verify project merge
	if len(merged.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(merged.Projects))
	}
	for _, p := range merged.Projects {
		if p.Name == "order-service" {
			if p.Revision != "develop" {
				t.Errorf("expected order-service revision=develop, got %q", p.Revision)
			}
			if p.Path != "services/order-service" {
				t.Errorf("expected path preserved, got %q", p.Path)
			}
		}
	}

	// Resolve
	resolved, syncJ, err := Resolve(merged)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if syncJ != 4 {
		t.Errorf("expected syncJ=4, got %d", syncJ)
	}
	for _, rp := range resolved {
		if !rp.HasPushRemote {
			t.Errorf("expected all projects to have push remote, %q doesn't", rp.Name)
		}
		if rp.CloneURL == "" {
			t.Errorf("expected non-empty clone URL for %q", rp.Name)
		}
		if rp.PushURL == "" {
			t.Errorf("expected non-empty push URL for %q", rp.Name)
		}
	}
}

func TestIntegrationOverlappingRemotes(t *testing.T) {
	base := &Manifest{
		Remotes: []Remote{
			{Name: "github", Fetch: "git@github.com:Org/", Review: "https://github.com/Org/"},
		},
		Default: &Default{Remote: "github", Revision: "master"},
	}
	local := &Manifest{
		Remotes: []Remote{
			{Name: "github", Fetch: "git@github.com:NewOrg/"},
		},
	}
	merged := Merge(base, local)
	if merged.Remotes[0].Fetch != "git@github.com:NewOrg/" {
		t.Errorf("expected overridden fetch, got %q", merged.Remotes[0].Fetch)
	}
	// Review should also be replaced (full replacement)
	if merged.Remotes[0].Review != "" {
		t.Errorf("expected review cleared by full replacement, got %q", merged.Remotes[0].Review)
	}
}

func TestIntegrationPartialDefaultOverrides(t *testing.T) {
	base := &Manifest{
		Default: &Default{Remote: "github", Revision: "master", SyncJ: "4", Push: "fork"},
	}
	local := &Manifest{
		Default: &Default{SyncJ: "16"},
	}
	merged := Merge(base, local)
	if merged.Default.Remote != "github" || merged.Default.Revision != "master" || merged.Default.Push != "fork" {
		t.Errorf("expected other attrs preserved, got %+v", merged.Default)
	}
	if merged.Default.SyncJ != "16" {
		t.Errorf("expected SyncJ=16, got %q", merged.Default.SyncJ)
	}
}

func TestIntegrationProjectAttributeInheritance(t *testing.T) {
	m := &Manifest{
		Remotes: []Remote{
			{Name: "github", Fetch: "git@github.com:Org/"},
			{Name: "fork", Fetch: "git@github.com:User/"},
			{Name: "custom", Fetch: "git@custom.com:Team/"},
		},
		Default: &Default{Remote: "github", Revision: "master", Push: "fork"},
		Projects: []Project{
			{Name: "svc-a", Path: "services/svc-a"},
			{Name: "svc-b", Path: "services/svc-b", Remote: "custom", Revision: "develop"},
			{Name: "svc-c", Path: "services/svc-c", Push: "custom"},
		},
	}

	resolved, _, err := Resolve(m)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// svc-a: inherits all defaults
	a := resolved[0]
	if a.Remote != "github" || a.Revision != "master" || a.Push != "fork" {
		t.Errorf("svc-a: expected full inheritance, got remote=%q revision=%q push=%q", a.Remote, a.Revision, a.Push)
	}

	// svc-b: overrides remote and revision, inherits push
	b := resolved[1]
	if b.Remote != "custom" || b.Revision != "develop" || b.Push != "fork" {
		t.Errorf("svc-b: expected overrides, got remote=%q revision=%q push=%q", b.Remote, b.Revision, b.Push)
	}

	// svc-c: overrides push, inherits remote and revision
	c := resolved[2]
	if c.Remote != "github" || c.Revision != "master" || c.Push != "custom" {
		t.Errorf("svc-c: expected push override, got remote=%q revision=%q push=%q", c.Remote, c.Revision, c.Push)
	}
	if c.PushURL != "git@custom.com:Team/svc-c.git" {
		t.Errorf("svc-c: unexpected push URL: %q", c.PushURL)
	}
}
