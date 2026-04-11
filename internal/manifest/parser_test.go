package manifest

import (
	"testing"
)

func TestParseValidManifest(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="github" fetch="git@github.com:Org/" review="https://github.com/Org/" />
  <remote name="fork" fetch="git@github.com:user/" />
  <default remote="github" revision="master" sync-j="8" push="fork" />
  <project name="my-service" path="services/my-service" groups="feed,be" remote="github" revision="develop" push="fork" />
  <project name="other-svc" path="services/other-svc" />
</manifest>`)

	m, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(m.Remotes))
	}
	if m.Remotes[0].Name != "github" || m.Remotes[0].Fetch != "git@github.com:Org/" {
		t.Errorf("unexpected first remote: %+v", m.Remotes[0])
	}
	if m.Remotes[0].Review != "https://github.com/Org/" {
		t.Errorf("expected review attribute, got %q", m.Remotes[0].Review)
	}
	if m.Remotes[1].Name != "fork" || m.Remotes[1].Review != "" {
		t.Errorf("unexpected second remote: %+v", m.Remotes[1])
	}

	if m.Default == nil {
		t.Fatal("expected default, got nil")
	}
	if m.Default.Remote != "github" || m.Default.Revision != "master" || m.Default.SyncJ != "8" || m.Default.Push != "fork" {
		t.Errorf("unexpected default: %+v", m.Default)
	}

	if len(m.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(m.Projects))
	}
	p := m.Projects[0]
	if p.Name != "my-service" || p.Path != "services/my-service" || p.Groups != "feed,be" || p.Revision != "develop" {
		t.Errorf("unexpected first project: %+v", p)
	}
	p2 := m.Projects[1]
	if p2.Name != "other-svc" || p2.Remote != "" || p2.Revision != "" {
		t.Errorf("unexpected second project: %+v", p2)
	}
}

func TestParseEmptyManifest(t *testing.T) {
	data := []byte(`<manifest></manifest>`)
	m, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d", len(m.Remotes))
	}
	if m.Default != nil {
		t.Errorf("expected nil default, got %+v", m.Default)
	}
	if len(m.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(m.Projects))
	}
}

func TestParseMalformedXML(t *testing.T) {
	data := []byte(`<manifest><broken`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for malformed XML")
	}
}

func TestParseDefaultWithoutOptionalAttrs(t *testing.T) {
	data := []byte(`<manifest><default remote="github" revision="master" /></manifest>`)
	m, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Default.SyncJ != "" {
		t.Errorf("expected empty SyncJ, got %q", m.Default.SyncJ)
	}
	if m.Default.Push != "" {
		t.Errorf("expected empty Push, got %q", m.Default.Push)
	}
}
