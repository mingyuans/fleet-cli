// Package selfupdate implements the logic behind the `fleet update` command:
// resolving the latest release, comparing versions, downloading and verifying
// release assets, and replacing the running binary in place.
//
// The pure helpers in this file (version comparison, asset naming, checksum
// verification, archive extraction) are deliberately free of network and
// filesystem side effects so they can be unit tested in isolation.
package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Repo is the GitHub repository that hosts fleet releases. It mirrors the
// REPO value baked into install.sh so both installers agree on the source.
const Repo = "mingyuans/fleet-cli"

// BinaryName is the name of the installed executable.
const BinaryName = "fleet"

// supportedOS and supportedArch capture the platform matrix produced by the
// release workflow. Anything outside these sets has no downloadable asset.
var (
	supportedOS   = map[string]bool{"darwin": true, "linux": true}
	supportedArch = map[string]bool{"amd64": true, "arm64": true}
)

// IsComparable reports whether version is a concrete, comparable release
// version. Development builds ("dev"/"unknown"/empty) cannot be compared
// against a remote tag and require --force to update.
func IsComparable(version string) bool {
	switch strings.TrimSpace(version) {
	case "", "dev", "unknown":
		return false
	default:
		_, ok := parseVersion(version)
		return ok
	}
}

// CompareVersions compares two version strings of the form "vX.Y.Z" (the
// leading "v" is optional). It returns -1 if a < b, 0 if a == b, 1 if a > b.
// ok is false when either version cannot be parsed (e.g. "dev" or a
// pre-release suffix), in which case the int result is meaningless.
func CompareVersions(a, b string) (cmp int, ok bool) {
	pa, okA := parseVersion(a)
	pb, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		switch {
		case x < y:
			return -1, true
		case x > y:
			return 1, true
		}
	}
	return 0, true
}

// parseVersion splits a "vX.Y.Z" string into its numeric segments. It returns
// ok=false if any segment is non-numeric (which also rejects pre-release
// suffixes such as "1.2.3-rc1").
func parseVersion(v string) ([]int, bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return nil, false
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		nums = append(nums, n)
	}
	return nums, true
}

// AssetName returns the release asset filename for the given platform, matching
// the naming convention produced by the release workflow
// (fleet-<os>-<arch>.tar.gz). It errors for unsupported os/arch combinations.
func AssetName(goos, goarch string) (string, error) {
	if !supportedOS[goos] || !supportedArch[goarch] {
		return "", fmt.Errorf("unsupported platform %s/%s (supported: darwin/linux, amd64/arm64)", goos, goarch)
	}
	return fmt.Sprintf("%s-%s-%s.tar.gz", BinaryName, goos, goarch), nil
}

// BinaryNameInArchive returns the name of the executable packaged inside the
// tar.gz for the given platform (fleet-<os>-<arch>).
func BinaryNameInArchive(goos, goarch string) string {
	return fmt.Sprintf("%s-%s-%s", BinaryName, goos, goarch)
}

// ParseChecksums parses the contents of a checksums.txt file (sha256sum
// format: "<hex>  <filename>") into a map of filename to lowercase hex digest.
func ParseChecksums(content []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// sha256sum marks binary mode with a leading '*' on the filename.
		name := strings.TrimPrefix(fields[1], "*")
		result[name] = strings.ToLower(fields[0])
	}
	return result
}

// VerifyChecksum confirms that data matches the SHA-256 recorded for assetName
// in the given checksums.txt content. It errors when the asset is absent from
// the checksum list or the digests differ.
func VerifyChecksum(data []byte, assetName string, checksums []byte) error {
	expected, ok := ParseChecksums(checksums)[assetName]
	if !ok {
		return fmt.Errorf("checksum for %s not found in checksums.txt", assetName)
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	return nil
}

// ExtractBinary reads a gzip-compressed tar archive and returns the contents of
// the first regular file it contains (the fleet executable).
func ExtractBinary(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read archived binary: %w", err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("no regular file found in archive")
}
