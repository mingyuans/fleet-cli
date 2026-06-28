package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		a, b    string
		wantCmp int
		wantOK  bool
	}{
		{"a less than b", "1.2.3", "1.2.4", -1, true},
		{"a greater than b", "1.3.0", "1.2.9", 1, true},
		{"equal", "1.2.3", "1.2.3", 0, true},
		{"v prefix equivalence", "v1.2.3", "1.2.3", 0, true},
		{"numeric not lexical", "1.10.0", "1.9.0", 1, true},
		{"different segment count", "1.2", "1.2.0", 0, true},
		{"different segment count newer", "1.2.1", "1.2", 1, true},
		{"dev not comparable", "dev", "1.2.3", 0, false},
		{"prerelease not comparable", "1.2.3-rc1", "1.2.3", 0, false},
		{"empty not comparable", "", "1.2.3", 0, false},
		{"garbage not comparable", "abc", "1.2.3", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmp, ok := CompareVersions(tt.a, tt.b)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && cmp != tt.wantCmp {
				t.Fatalf("cmp = %d, want %d", cmp, tt.wantCmp)
			}
		})
	}
}

func TestIsComparable(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"1.2.3", true},
		{"v1.2.3", true},
		{"dev", false},
		{"unknown", false},
		{"", false},
		{"1.2.3-rc1", false},
	}
	for _, tt := range tests {
		if got := IsComparable(tt.version); got != tt.want {
			t.Errorf("IsComparable(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
		wantErr      bool
	}{
		{"darwin", "arm64", "fleet-darwin-arm64.tar.gz", false},
		{"darwin", "amd64", "fleet-darwin-amd64.tar.gz", false},
		{"linux", "amd64", "fleet-linux-amd64.tar.gz", false},
		{"linux", "arm64", "fleet-linux-arm64.tar.gz", false},
		{"windows", "amd64", "", true},
		{"darwin", "386", "", true},
		{"plan9", "ppc64", "", true},
	}
	for _, tt := range tests {
		got, err := AssetName(tt.goos, tt.goarch)
		if tt.wantErr {
			if err == nil {
				t.Errorf("AssetName(%q, %q) expected error, got nil", tt.goos, tt.goarch)
			}
			continue
		}
		if err != nil {
			t.Errorf("AssetName(%q, %q) unexpected error: %v", tt.goos, tt.goarch, err)
		}
		if got != tt.want {
			t.Errorf("AssetName(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("hello fleet")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])

	checksums := []byte(hexSum + "  fleet-darwin-arm64.tar.gz\n" +
		"deadbeef  fleet-linux-amd64.tar.gz\n")

	t.Run("matching checksum", func(t *testing.T) {
		if err := VerifyChecksum(data, "fleet-darwin-arm64.tar.gz", checksums); err != nil {
			t.Fatalf("expected match, got error: %v", err)
		}
	})

	t.Run("mismatched checksum", func(t *testing.T) {
		if err := VerifyChecksum(data, "fleet-linux-amd64.tar.gz", checksums); err == nil {
			t.Fatal("expected mismatch error, got nil")
		}
	})

	t.Run("asset not in checksums", func(t *testing.T) {
		if err := VerifyChecksum(data, "fleet-linux-arm64.tar.gz", checksums); err == nil {
			t.Fatal("expected not-found error, got nil")
		}
	})
}

func TestParseChecksums(t *testing.T) {
	content := []byte("abc123  fleet-darwin-arm64.tar.gz\n" +
		"def456 *fleet-linux-amd64.tar.gz\n" +
		"\n" +
		"  \n" +
		"incompleteline\n")
	got := ParseChecksums(content)
	if got["fleet-darwin-arm64.tar.gz"] != "abc123" {
		t.Errorf("darwin checksum = %q, want abc123", got["fleet-darwin-arm64.tar.gz"])
	}
	// The leading '*' (binary mode marker) must be stripped from the filename.
	if got["fleet-linux-amd64.tar.gz"] != "def456" {
		t.Errorf("linux checksum = %q, want def456", got["fleet-linux-amd64.tar.gz"])
	}
	if len(got) != 2 {
		t.Errorf("parsed %d entries, want 2", len(got))
	}
}

func TestExtractBinary(t *testing.T) {
	want := []byte("#!/bin/sh\necho fleet\n")
	archive := makeTarGz(t, "fleet-darwin-arm64", want)

	got, err := ExtractBinary(archive)
	if err != nil {
		t.Fatalf("ExtractBinary error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("extracted = %q, want %q", got, want)
	}
}

func TestExtractBinaryInvalidArchive(t *testing.T) {
	if _, err := ExtractBinary([]byte("not a gzip stream")); err == nil {
		t.Fatal("expected error for invalid archive, got nil")
	}
}

// makeTarGz builds an in-memory gzip-compressed tar containing a single file.
func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}
