package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "poiley"
	repoName  = "beady"
)

// githubRelease represents a GitHub release API response.
type githubRelease struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Update checks for a newer version and installs it.
func Update(currentVersion string) error {
	fmt.Println("Checking for updates...")

	release, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if latestVersion == currentClean {
		fmt.Printf("Already up to date (v%s).\n", currentClean)
		return nil
	}

	fmt.Printf("New version available: v%s -> v%s\n", currentClean, latestVersion)

	// Find the right asset for this OS/arch
	assetName := getAssetName()
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		// Fallback: try go install
		fmt.Println("No pre-built binary found for your platform. Trying go install...")
		return goInstall(release.TagName)
	}

	// Download and replace
	return downloadAndInstall(downloadURL, latestVersion)
}

func getLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("no releases found. Repository %s/%s may not have any releases yet", repoOwner, repoName)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func getAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Match goreleaser naming convention
	switch os {
	case "darwin":
		os = "darwin"
	case "linux":
		os = "linux"
	case "windows":
		os = "windows"
	}

	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	}

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("bdy_%s_%s%s", os, arch, ext)
}

func downloadAndInstall(url, version string) error {
	fmt.Printf("Downloading v%s...\n", version)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "bdy-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("downloading: %w", err)
	}
	tmpFile.Close()

	// Extract the binary
	binPath, err := extractBinary(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("extracting: %w", err)
	}
	defer os.Remove(binPath)

	// Find current binary location
	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	currentBin, err = filepath.EvalSymlinks(currentBin)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	// Replace current binary
	if err := replaceBinary(binPath, currentBin); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Printf("Updated to v%s.\n", version)
	return nil
}

func extractBinary(archivePath string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "bdy-extract-*")
	if err != nil {
		return "", err
	}

	// Use tar to extract
	cmd := exec.Command("tar", "xzf", archivePath, "-C", tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("tar extract failed: %s: %w", string(out), err)
	}

	// Find the bdy binary in extracted files
	binName := "bdy"
	if runtime.GOOS == "windows" {
		binName = "bdy.exe"
	}

	binPath := filepath.Join(tmpDir, binName)
	if _, err := os.Stat(binPath); err != nil {
		// Try looking in subdirectories (goreleaser sometimes nests)
		entries, _ := os.ReadDir(tmpDir)
		for _, e := range entries {
			if e.IsDir() {
				nested := filepath.Join(tmpDir, e.Name(), binName)
				if _, err := os.Stat(nested); err == nil {
					binPath = nested
					break
				}
			}
		}
	}

	if _, err := os.Stat(binPath); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("binary not found in archive")
	}

	return binPath, nil
}

func replaceBinary(newBin, currentBin string) error {
	// Make new binary executable
	if err := os.Chmod(newBin, 0755); err != nil {
		return err
	}

	// Atomic replace: rename new over old
	// On Unix, this works even if the binary is running
	if err := os.Rename(newBin, currentBin); err != nil {
		// Cross-device rename; fall back to copy
		return copyFile(newBin, currentBin)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Write to temp next to dst, then rename
	tmpDst := dst + ".new"
	out, err := os.OpenFile(tmpDst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmpDst)
		return err
	}
	out.Close()

	return os.Rename(tmpDst, dst)
}

func goInstall(tag string) error {
	ref := tag
	if ref == "" {
		ref = "latest"
	}
	cmd := exec.Command("go", "install", fmt.Sprintf("github.com/%s/%s/cmd/bdy@%s", repoOwner, repoName, ref))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}
	fmt.Printf("Updated via go install to %s.\n", ref)
	return nil
}
