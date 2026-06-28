package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// httpTimeout bounds each network request so the command never hangs forever.
const httpTimeout = 30 * time.Second

// LatestVersion queries the GitHub Releases API for the most recent published
// release of Repo and returns its tag_name.
func LatestVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", Repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %s when fetching latest release", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("latest release has no tag_name")
	}
	return payload.TagName, nil
}

// downloadURL builds the public download URL for a release asset.
func downloadURL(tag, assetName string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", Repo, tag, assetName)
}

// DownloadAsset downloads the release asset and the accompanying checksums.txt
// for the given tag, returning their raw bytes.
func DownloadAsset(ctx context.Context, tag, assetName string) (asset []byte, checksums []byte, err error) {
	asset, err = download(ctx, downloadURL(tag, assetName))
	if err != nil {
		return nil, nil, fmt.Errorf("download %s: %w", assetName, err)
	}
	checksums, err = download(ctx, downloadURL(tag, "checksums.txt"))
	if err != nil {
		return nil, nil, fmt.Errorf("download checksums.txt: %w", err)
	}
	return asset, checksums, nil
}

// download fetches a URL and returns its body, erroring on non-2xx responses.
func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
