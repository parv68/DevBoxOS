package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade DevBoxOS to the latest version",
	Long:  `Check for updates and download the latest DevBoxOS binaries from GitHub Releases.`,
	RunE:  runUpgrade,
}

var (
	upgradeVersion string
	upgradeDryRun  bool
)

func init() {
	upgradeCmd.Flags().StringVar(&upgradeVersion, "version", "latest", "Version to upgrade to")
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Check for updates without downloading")
	rootCmd.AddCommand(upgradeCmd)
}

type GitHubRelease struct {
	TagName    string         `json:"tag_name"`
	Assets     []GitHubAsset  `json:"assets"`
	Prerelease bool           `json:"prerelease"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	currentVersion := getVersion()
	info, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(info.TagName, "v")

	if currentVersion == latestVersion && upgradeVersion == "latest" {
		fmt.Printf("✓ DevBoxOS is already up to date (%s)\n", currentVersion)
		return nil
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest version:  %s\n", latestVersion)

	if upgradeDryRun {
		fmt.Println("\nDry run - no changes made")
		return nil
	}

	fmt.Println("\nUpgrading...")

	if err := downloadAndInstall(info); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Printf("✓ Upgraded to %s\n", latestVersion)
	return nil
}

func getVersion() string {
	// This would normally come from build-time ldflags
	return "0.1.0-dev"
}

func getLatestRelease() (*GitHubRelease, error) {
	url := "https://api.github.com/repos/devboxos/devboxos/releases/latest"
	if upgradeVersion != "latest" {
		url = fmt.Sprintf("https://api.github.com/repos/devboxos/devboxos/releases/tags/%s", upgradeVersion)
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadAndInstall(release *GitHubRelease) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	}

	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}

	cliAsset := fmt.Sprintf("devbox-%s-%s-%s%s", release.TagName, osName, arch, ext)
	engineAsset := fmt.Sprintf("devbox-engine-%s-%s-%s%s", release.TagName, osName, arch, ext)

	cliURL := findAssetURL(release, cliAsset)
	engineURL := findAssetURL(release, engineAsset)

	if cliURL == "" {
		return fmt.Errorf("binary not found for %s/%s", osName, arch)
	}

	// Download CLI binary
	tmpFile, err := os.CreateTemp("", "devbox-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if err := downloadFile(cliURL, tmpFile); err != nil {
		return err
	}
	tmpFile.Close()

	// Install CLI binary
	destPath, err := os.Executable()
	if err != nil {
		destPath = filepath.Join("/usr/local/bin", "devboxos"+ext)
	}

	if err := os.Rename(tmpFile.Name(), destPath); err != nil {
		// Fallback: copy if rename fails
		if err := copyFile(tmpFile.Name(), destPath); err != nil {
			return err
		}
	}

	// Download engine binary
	if engineURL != "" {
		engineTmpFile, err := os.CreateTemp("", "devbox-engine-*")
		if err != nil {
			return err
		}
		defer os.Remove(engineTmpFile.Name())

		if err := downloadFile(engineURL, engineTmpFile); err != nil {
			fmt.Printf("Warning: failed to download engine: %v\n", err)
		} else {
			engineTmpFile.Close()
			engineDest := filepath.Join(filepath.Dir(destPath), "devbox-engine"+ext)
			os.Rename(engineTmpFile.Name(), engineDest)
		}
	}

	return nil
}

func findAssetURL(release *GitHubRelease, name string) string {
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

func downloadFile(url string, dst *os.File) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(dst, resp.Body)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		return out.Chmod(0755)
	}
	return nil
}
