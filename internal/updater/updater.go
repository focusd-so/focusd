package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	selfupdate "github.com/creativeprojects/go-selfupdate"
)

const (
	startupDelay    = 10 * time.Second
	checkInterval   = 6 * time.Hour
	zipAssetName    = "Focusd.zip"
	sha256AssetName = "Focusd.zip.sha256"
	maxDownloadSize = 200 << 20 // 200 MB
)

var allowedDownloadHosts = map[string]bool{
	"github.com":                            true,
	"objects.githubusercontent.com":         true,
	"github-releases.githubusercontent.com": true,
}

type UpdateInfo struct {
	Version      string `json:"version"`
	ReleaseNotes string `json:"releaseNotes"`
}

type Service struct {
	currentVersion *semver.Version
	repo           selfupdate.RepositorySlug
	source         *selfupdate.GitHubSource
	teamID         string
}

func NewService(version, owner, repo, teamID string) *Service {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		slog.Error("failed to create GitHub source for updater", "error", err)
		return nil
	}

	v, err := semver.NewVersion(strings.TrimPrefix(version, "v"))
	if err != nil {
		slog.Warn("invalid version for updater, updates disabled", "version", version, "error", err)
		return nil
	}

	return &Service{
		currentVersion: v,
		repo:           selfupdate.NewRepositorySlug(owner, repo),
		source:         source,
		teamID:         teamID,
	}
}

func (s *Service) GetCurrentVersion() string {
	return s.currentVersion.String()
}

// CheckForUpdate queries GitHub for the latest release and returns update info
// if a newer version is available.
func (s *Service) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	rel, _, _, err := s.findLatestRelease(ctx)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, nil
	}
	return &UpdateInfo{
		Version:      rel.GetTagName(),
		ReleaseNotes: rel.GetReleaseNotes(),
	}, nil
}

// ApplyUpdate checks for and applies the latest update immediately (used by
// the manual "Check for Updates" tray menu item).
func (s *Service) ApplyUpdate(ctx context.Context) {
	if err := s.checkAndApply(ctx); err != nil {
		slog.Error("manual update failed", "error", err)
	}
}

// Start runs the silent background update loop. It checks for updates after an
// initial delay and then every checkInterval. When a newer version is found it
// downloads, replaces the .app bundle, and restarts.
func (s *Service) Start(ctx context.Context) {
	select {
	case <-time.After(startupDelay):
	case <-ctx.Done():
		return
	}

	for {
		if err := s.checkAndApply(ctx); err != nil {
			slog.Error("auto-update check failed", "error", err)
		}

		select {
		case <-time.After(checkInterval):
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) checkAndApply(ctx context.Context) error {
	slog.Info("checking for updates", "current", s.currentVersion)

	rel, zipAsset, sha256Asset, err := s.findLatestRelease(ctx)
	if err != nil {
		return err
	}
	if rel == nil {
		slog.Info("already up to date")
		return nil
	}

	slog.Info("new version available", "latest", rel.GetTagName(), "current", s.currentVersion)

	appPath, err := resolveAppPath()
	if err != nil {
		return fmt.Errorf("resolving .app path: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "focusd-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, zipAssetName)
	if err := downloadFile(ctx, zipAsset.GetBrowserDownloadURL(), zipPath); err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}

	if sha256Asset != nil {
		if err := verifySHA256(ctx, zipPath, sha256Asset.GetBrowserDownloadURL()); err != nil {
			return fmt.Errorf("checksum verification: %w", err)
		}
		slog.Info("SHA256 checksum verified")
	} else {
		slog.Warn("no SHA256 checksum asset found, skipping checksum verification")
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("creating extract dir: %w", err)
	}
	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extracting update: %w", err)
	}

	newAppPath := filepath.Join(extractDir, filepath.Base(appPath))
	if _, err := os.Stat(newAppPath); os.IsNotExist(err) {
		return fmt.Errorf("extracted app not found at %s", newAppPath)
	}

	if s.teamID != "" {
		if err := verifyCodeSignature(newAppPath, s.teamID); err != nil {
			return fmt.Errorf("code signature verification: %w", err)
		}
		slog.Info("code signature verified", "teamID", s.teamID)
	} else {
		slog.Warn("no team ID configured, skipping code signature verification")
	}

	if err := replaceApp(appPath, newAppPath); err != nil {
		return fmt.Errorf("replacing app bundle: %w", err)
	}

	slog.Info("update applied successfully, restarting", "version", rel.GetTagName())
	return relaunch(appPath)
}

// findLatestRelease returns the newest non-draft, non-prerelease release that
// is newer than the current version and contains a Focusd.zip asset. Returns
// nil values when already up to date.
func (s *Service) findLatestRelease(ctx context.Context) (selfupdate.SourceRelease, selfupdate.SourceAsset, selfupdate.SourceAsset, error) {
	releases, err := s.source.ListReleases(ctx, s.repo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("listing releases: %w", err)
	}

	slog.Info("updater: found releases", "count", len(releases), "current", s.currentVersion)

	for _, rel := range releases {
		tag := rel.GetTagName()
		slog.Info("updater: checking release", "tag", tag, "draft", rel.GetDraft(), "prerelease", rel.GetPrerelease())

		if rel.GetDraft() || rel.GetPrerelease() {
			continue
		}

		v, err := semver.NewVersion(strings.TrimPrefix(tag, "v"))
		if err != nil {
			slog.Warn("updater: skipping release with unparseable tag", "tag", tag, "error", err)
			continue
		}

		slog.Info("updater: version check", "release", v, "current", s.currentVersion, "newer", v.GreaterThan(s.currentVersion))

		if !v.GreaterThan(s.currentVersion) {
			slog.Info("updater: already up to date")
			return nil, nil, nil, nil
		}

		var zipFound selfupdate.SourceAsset
		var sha256Found selfupdate.SourceAsset
		var assetNames []string
		for _, asset := range rel.GetAssets() {
			assetNames = append(assetNames, asset.GetName())
			switch asset.GetName() {
			case zipAssetName:
				zipFound = asset
			case sha256AssetName:
				sha256Found = asset
			}
		}

		if zipFound != nil {
			slog.Info("updater: found zip asset", "tag", tag, "hasSHA256", sha256Found != nil)
			return rel, zipFound, sha256Found, nil
		}
		slog.Warn("updater: newer release has no zip asset, skipping", "tag", tag, "assets", assetNames)
	}

	return nil, nil, nil, nil
}

// resolveAppPath walks up from the running executable to find the .app bundle root.
// e.g. /Applications/Focusd.app/Contents/MacOS/Focusd -> /Applications/Focusd.app
func resolveAppPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(exe)
	for i := 0; i < 5; i++ {
		if strings.HasSuffix(dir, ".app") {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("could not find .app bundle from executable path %s", exe)
}

var httpClient = &http.Client{
	Timeout: 5 * time.Minute,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return fmt.Errorf("too many redirects")
		}
		if !allowedDownloadHosts[req.URL.Hostname()] {
			return fmt.Errorf("redirect to untrusted host: %s", req.URL.Hostname())
		}
		return nil
	},
}

func downloadFile(ctx context.Context, url, dest string) error {
	slog.Info("downloading update", "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, io.LimitReader(resp.Body, maxDownloadSize))
	if err != nil {
		return err
	}
	if n >= maxDownloadSize {
		return fmt.Errorf("download exceeds maximum allowed size of %d bytes", maxDownloadSize)
	}
	return f.Close()
}

func verifySHA256(ctx context.Context, filePath, checksumURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching checksum: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return err
	}

	// Format: "<hex>  filename" or just "<hex>"
	expectedHash := strings.TrimSpace(strings.Fields(string(body))[0])

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

func verifyCodeSignature(appPath, expectedTeamID string) error {
	verify := exec.Command("codesign", "--verify", "--deep", "--strict", appPath)
	if out, err := verify.CombinedOutput(); err != nil {
		return fmt.Errorf("signature invalid: %w: %s", err, out)
	}

	info := exec.Command("codesign", "-dvv", appPath)
	out, err := info.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reading signing info: %w: %s", err, out)
	}
	if !strings.Contains(string(out), "TeamIdentifier="+expectedTeamID) {
		return fmt.Errorf("team ID mismatch in downloaded app, expected %s", expectedTeamID)
	}
	return nil
}

func extractZip(zipPath, destDir string) error {
	cmd := exec.Command("ditto", "-xk", zipPath, destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ditto: %w: %s", err, out)
	}
	return nil
}

func replaceApp(currentApp, newApp string) error {
	backupPath := currentApp + ".old"

	os.RemoveAll(backupPath)

	if err := os.Rename(currentApp, backupPath); err != nil {
		return fmt.Errorf("backing up current app: %w", err)
	}

	if err := os.Rename(newApp, currentApp); err != nil {
		if rbErr := os.Rename(backupPath, currentApp); rbErr != nil {
			slog.Error("rollback failed", "error", rbErr)
		}
		return fmt.Errorf("moving new app into place: %w", err)
	}

	os.RemoveAll(backupPath)
	return nil
}

func relaunch(appPath string) error {
	cmd := exec.Command("open", "-n", appPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("relaunching: %w", err)
	}
	os.Exit(0)
	return nil // unreachable
}
