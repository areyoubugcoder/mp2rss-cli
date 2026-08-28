// Package update implements `mp2rss update` self-updating the installed binary.
//
// Flow:
//  1. Query GitHub Releases API for the latest tag.
//  2. Compare with the build-time injected version. --check exits here.
//  3. Pick the platform archive (tar.gz / zip), download to a temp dir.
//  4. Download checksums.txt and verify SHA-256.
//  5. Extract the binary, stage it at <selfpath>.new, then os.Rename atomically.
//  6. macOS only: strip the Gatekeeper quarantine xattr so the binary runs.
//
// `os.Executable()` plus `filepath.EvalSymlinks` resolves the path so users
// installed via Homebrew / npm postinstall symlinks still get updated in place.
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/skills"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
	"github.com/spf13/cobra"
)

// Repo is the GitHub slug we pull releases from. Exposed as a var so tests
// (and future enterprise builds) can repoint it.
var Repo = "areyoubugcoder/mp2rss-cli"

// LatestReleaseURL is overridable so tests can point at httptest servers.
var LatestReleaseURL = func() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", Repo)
}

// DownloadBase is overridable so tests can point at httptest servers.
var DownloadBase = func() string {
	return fmt.Sprintf("https://github.com/%s/releases/download", Repo)
}

// NewCmd returns the `mp2rss update` command.
func NewCmd() *cobra.Command {
	var (
		flagCheck      bool
		flagForce      bool
		flagSkipSkills bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "自更新到 GitHub 上的最新版本",
		Long: `检查 GitHub Releases 中的最新 mp2rss-cli 版本：
* --check: 仅查询不下载，已是最新返回退出码 0，有新版本返回 0 并打印；
* --force: 即使本地版本与远端一致也强制重新下载替换；
* --skip-skills: 本次更新不同步本地 Agent Skills；
* 默认: 有新版本则下载、校验 checksums.txt（SHA-256）、原子替换当前二进制，
  并在之前通过 mp2rss skills sync 装过 skills 时顺带同步到新版本。

通过 npm / Homebrew / 包管理器安装的用户更建议用对应方式更新（avoid 自身覆盖）。`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			return run(out, flagCheck, flagForce, flagSkipSkills)
		},
	}
	cmd.Flags().BoolVar(&flagCheck, "check", false, "仅检查是否有新版本，不下载")
	cmd.Flags().BoolVar(&flagForce, "force", false, "强制重新下载并替换，即使版本相同")
	cmd.Flags().BoolVar(&flagSkipSkills, "skip-skills", false, "本次更新不同步本地 Agent Skills")
	return cmd
}

func run(out io.Writer, check, force, skipSkills bool) error {
	fmt.Fprintln(out, "🔎 正在查询最新版本…")
	latest, err := fetchLatestTag()
	if err != nil {
		return fmt.Errorf("查询最新版本失败：%w", err)
	}
	current := version.Version
	cmp := version.Compare(current, latest)

	if check {
		switch {
		case cmp >= 0:
			fmt.Fprintf(out, "✓ 当前版本 %s 已是最新（远端 %s）\n", current, latest)
		default:
			fmt.Fprintf(out, "⬆ 有新版本：%s → %s\n", current, latest)
			fmt.Fprintln(out, "  运行 `mp2rss update` 完成升级。")
		}
		return nil
	}
	if cmp >= 0 && !force {
		fmt.Fprintf(out, "✓ 当前版本 %s 已是最新，无需更新（使用 --force 强制重装）\n", current)
		return nil
	}
	if force && cmp >= 0 {
		fmt.Fprintf(out, "⚙ 强制重装 %s\n", latest)
	} else {
		fmt.Fprintf(out, "⬆ 升级 %s → %s\n", current, latest)
	}

	if err := doUpdate(out, latest); err != nil {
		return err
	}
	if !skipSkills {
		maybeSyncSkills(out, latest)
	}
	return nil
}

// maybeSyncSkills follows up a binary update by syncing local Agent Skills to
// the new version — but only for users who previously opted in by running
// `mp2rss skills sync` (a state file exists). Failures are non-fatal: the
// binary update itself already succeeded.
func maybeSyncSkills(out io.Writer, latest string) {
	st := skills.ReadState()
	if st == nil {
		fmt.Fprintln(out, "ℹ 使用 Agent Skills 的话，可运行 `mp2rss skills sync` 同步到新版本。")
		return
	}
	if version.Compare(st.Version, latest) == 0 {
		return
	}
	fmt.Fprintf(out, "⏳ 同步本地 skills 到 %s（%s）…\n", latest, st.Scope)
	res, err := skills.Sync(st.Scope, latest)
	if err != nil {
		fmt.Fprintf(out, "⚠ skills 同步失败（不影响本次更新）：%v\n  可稍后手动运行 `mp2rss skills sync`。\n", err)
		return
	}
	fmt.Fprintf(out, "✓ 已同步 %d 个 skills 到 %s\n", len(res.Skills), res.Version)
}

func doUpdate(out io.Writer, latest string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if err := supported(goos, goarch); err != nil {
		return err
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	asset := version.AssetName(latest, goos, goarch, ext)
	dlURL := fmt.Sprintf("%s/%s/%s", DownloadBase(), latest, asset)
	sumsURL := fmt.Sprintf("%s/%s/checksums.txt", DownloadBase(), latest)

	tmpDir, err := os.MkdirTemp("", "mp2rss-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	fmt.Fprintf(out, "  ↓ 下载 %s\n", asset)
	archivePath := filepath.Join(tmpDir, asset)
	if err := download(dlURL, archivePath); err != nil {
		return fmt.Errorf("下载失败：%w", err)
	}

	fmt.Fprintln(out, "  🔐 校验 SHA-256")
	sumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := download(sumsURL, sumsPath); err != nil {
		return fmt.Errorf("下载 checksums.txt 失败：%w", err)
	}
	expected, err := lookupChecksum(sumsPath, asset)
	if err != nil {
		return err
	}
	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum 不匹配（asset=%s）: expected %s, got %s", asset, expected, actual)
	}

	binaryName := "mp2rss"
	if goos == "windows" {
		binaryName = "mp2rss.exe"
	}
	extractedPath := filepath.Join(tmpDir, binaryName)
	if err := extractBinary(archivePath, ext, binaryName, extractedPath); err != nil {
		return fmt.Errorf("解压失败：%w", err)
	}

	selfPath, err := resolveSelfPath()
	if err != nil {
		return fmt.Errorf("定位自身可执行文件：%w", err)
	}
	if err := atomicReplace(selfPath, extractedPath); err != nil {
		return fmt.Errorf("原子替换：%w", err)
	}
	if goos == "darwin" {
		stripQuarantine(selfPath)
	}

	fmt.Fprintf(out, "✓ 已更新到 %s（位置：%s）\n", latest, selfPath)
	return nil
}

// fetchLatestTag returns the tag_name of the latest release.
func fetchLatestTag() (string, error) {
	c := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, LatestReleaseURL(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mp2rss-cli")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", errors.New("GitHub API 返回空 tag_name")
	}
	return body.TagName, nil
}

func supported(goos, goarch string) error {
	switch goos {
	case "darwin", "linux", "windows":
	default:
		return fmt.Errorf("当前平台暂未提供预编译产物：%s/%s", goos, goarch)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("当前架构暂未提供预编译产物：%s/%s", goos, goarch)
	}
	return nil
}

func download(url, dst string) error {
	c := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mp2rss-cli")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// LookupChecksum returns the hex digest recorded in a `sha256sum`-style
// checksums.txt file for the given filename. Exposed for testing.
func LookupChecksum(path, filename string) (string, error) {
	return lookupChecksum(path, filename)
}

func lookupChecksum(path, filename string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: "<sha256>  <filename>" or "<sha256> *<filename>"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		// match either bare name or any path ending with /<name>
		if name == filename || strings.HasSuffix(name, "/"+filename) {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksums.txt 中找不到 %s", filename)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractBinary(archive, ext, binaryName, dst string) error {
	if ext == ".zip" {
		return extractZip(archive, binaryName, dst)
	}
	return extractTarGz(archive, binaryName, dst)
}

func extractTarGz(archive, binaryName, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, tr); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("归档中未找到 %s", binaryName)
}

func extractZip(archive, binaryName, dst string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, rc)
		return err
	}
	return fmt.Errorf("ZIP 中未找到 %s", binaryName)
}

func resolveSelfPath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err == nil {
		return resolved, nil
	}
	// On some platforms EvalSymlinks fails when path is already canonical.
	return p, nil
}

// atomicReplace stages a new binary alongside `selfPath` and renames it in.
// On Windows, in-place os.Rename can fail when the running process holds the
// file open — we then move the old file aside first.
func atomicReplace(selfPath, newBinary string) error {
	stagingPath := selfPath + ".new"
	if err := copyFile(newBinary, stagingPath); err != nil {
		return err
	}
	if err := os.Chmod(stagingPath, 0o755); err != nil {
		_ = os.Remove(stagingPath)
		return err
	}
	if err := os.Rename(stagingPath, selfPath); err == nil {
		return nil
	}
	// Windows fallback: move current binary aside, then rename.
	oldPath := selfPath + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(selfPath, oldPath); err != nil {
		_ = os.Remove(stagingPath)
		return err
	}
	if err := os.Rename(stagingPath, selfPath); err != nil {
		_ = os.Rename(oldPath, selfPath)
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// stripQuarantine best-effort removes macOS Gatekeeper quarantine attribute
// that's applied to anything downloaded by HTTP. Failures are non-fatal.
func stripQuarantine(path string) {
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", path)
	_ = cmd.Run()
}
