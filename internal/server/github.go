package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuyuanzhang1122/bililive-server-update/internal/model"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	UpdatedAt          string `json:"updated_at"`
}

func fetchLatestGitHubRelease(ctx context.Context, repo string) ([]model.ReleaseArtifact, error) {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if repo == "" || !strings.Contains(repo, "/") {
		return nil, fmt.Errorf("GitHub repo 配置非法: %q", repo)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bililive-server-update")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub release API 返回 %s", resp.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	items := make([]model.ReleaseArtifact, 0, len(release.Assets)+1)
	if len(release.Assets) == 0 {
		items = append(items, model.ReleaseArtifact{
			Source:      "github",
			Name:        release.Name,
			Version:     release.TagName,
			Channel:     "stable",
			InstallMode: "source",
			URL:         release.HTMLURL,
			UpdatedAt:   now,
		})
		return items, nil
	}
	for _, asset := range release.Assets {
		updatedAt := now
		if parsed, err := time.Parse(time.RFC3339, asset.UpdatedAt); err == nil {
			updatedAt = parsed
		}
		osName, arch := inferOSArch(asset.Name)
		items = append(items, model.ReleaseArtifact{
			Source:      "github",
			Name:        asset.Name,
			Version:     release.TagName,
			Channel:     "stable",
			InstallMode: inferInstallMode(asset.Name),
			OS:          osName,
			Arch:        arch,
			URL:         asset.BrowserDownloadURL,
			Size:        asset.Size,
			UpdatedAt:   updatedAt,
			Metadata: map[string]string{
				"release_url": release.HTMLURL,
			},
		})
	}
	return items, nil
}

func inferInstallMode(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "docker"):
		return "docker"
	case strings.HasSuffix(lower, ".tgz"):
		return "npm"
	default:
		return "binary"
	}
}

func inferOSArch(name string) (string, string) {
	lower := strings.ToLower(filepath.Base(name))
	osName := ""
	arch := ""
	for _, candidate := range []string{"darwin", "macos", "linux", "windows", "win"} {
		if strings.Contains(lower, candidate) {
			osName = candidate
			break
		}
	}
	if osName == "macos" {
		osName = "darwin"
	}
	if osName == "win" {
		osName = "windows"
	}
	for _, candidate := range []string{"amd64", "x86_64", "arm64", "aarch64", "386"} {
		if strings.Contains(lower, candidate) {
			arch = candidate
			break
		}
	}
	if arch == "x86_64" {
		arch = "amd64"
	}
	if arch == "aarch64" {
		arch = "arm64"
	}
	return osName, arch
}
