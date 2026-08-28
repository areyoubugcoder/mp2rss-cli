// Package skills syncs the repo's Agent Skills to the local agent config.
//
// Skills (skills/mp2rss-*) ship with the CLI repo but are installed separately
// into ~/.claude/skills (global) or ./.agents/skills (project) via the
// community `skills` CLI (https://github.com/vercel-labs/skills). A CLI
// self-update only replaces the binary, so local skills drift behind. This
// package provides:
//
//   - Sync: run `npx -y skills add <repo> [-g] -y` and record the reached CLI
//     version in a state file (~/.mp2rss/skills-state.json);
//   - ReadState / Drift: detect "binary updated but local skills stale".
package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/areyoubugcoder/mp2rss-cli/internal/errs"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
)

// RepoSlug is the GitHub repo the skills are pulled from. Var so tests (and
// future forks) can repoint it.
var RepoSlug = "areyoubugcoder/mp2rss-cli"

// Scope is where skills get installed.
type Scope string

// Install scopes: global → ~/.claude/skills (all projects); project → the
// current directory's ./.agents/skills.
const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

// Skill names one official skill with a one-line description for listings.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Official lists the skills this repo provides. Display-only: the actual sync
// installs all repo skills via `skills add`. Keep in sync with skills/.
var Official = []Skill{
	{Name: "mp2rss-auth", Description: "登录 / 登出 / 查看登录态（auth login / logout / status）"},
	{Name: "mp2rss-mp", Description: "公众号订阅与文章（mp subscribe / list / search / remove / articles）"},
	{Name: "mp2rss-x", Description: "X 账号只读：已订阅列表 / 推文流 / 长文流（x list / posts / articles）"},
}

// State is the on-disk sync state, the sole basis for drift detection.
type State struct {
	// Version is the CLI version local skills were last synced to.
	Version string `json:"version"`
	// Skills are the synced skill names.
	Skills []string `json:"skills"`
	// Scope is the install scope of the last sync.
	Scope Scope `json:"scope"`
	// SyncedAt is the sync time, ISO 8601 UTC.
	SyncedAt string `json:"syncedAt"`
}

// StatePath returns the state file path. Var so tests can point elsewhere.
var StatePath = func() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mp2rss", "skills-state.json"), nil
}

// ReadState loads the state file. Missing or corrupt files yield nil ("never
// synced via mp2rss") so cold starts stay quiet.
func ReadState() *State {
	path, err := StatePath()
	if err != nil {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var st State
	if json.Unmarshal(raw, &st) != nil || st.Version == "" {
		return nil
	}
	if st.Scope != ScopeGlobal {
		st.Scope = ScopeProject
	}
	return &st
}

// WriteState persists the state file (0700 dir / 0600 file, like config.json).
func WriteState(st State) error {
	path, err := StatePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dir, 0o700)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// BuildArgs assembles the `npx` argument list for a sync.
func BuildArgs(scope Scope) []string {
	args := []string{"-y", "skills", "add", RepoSlug, "-y"}
	if scope == ScopeGlobal {
		args = append(args, "-g")
	}
	return args
}

// Runner executes npx with the given args and returns combined output.
// Injectable so tests avoid spawning processes / hitting the network.
var Runner = func(args []string) (string, error) {
	cmd := exec.Command("npx", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Result describes a completed sync.
type Result struct {
	Version string   `json:"version"`
	Skills  []string `json:"skills"`
	Scope   Scope    `json:"scope"`
}

// Sync installs/updates the local skills via `npx skills add` and writes the
// state file with toVersion on success.
func Sync(scope Scope, toVersion string) (*Result, error) {
	out, err := Runner(BuildArgs(scope))
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errs.Newf(errs.CodeGeneric,
				"未找到 npx（需要 Node.js）。请安装 Node.js 后重试，或手动安装：npx -y skills add %s -y", RepoSlug)
		}
		detail := out
		if len(detail) > 2000 {
			detail = detail[:2000]
		}
		return nil, errs.Newf(errs.CodeGeneric, "同步 skills 失败：%v\n%s", err, detail)
	}
	names := make([]string, len(Official))
	for i, s := range Official {
		names[i] = s.Name
	}
	st := State{
		Version:  toVersion,
		Skills:   names,
		Scope:    scope,
		SyncedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := WriteState(st); err != nil {
		return nil, fmt.Errorf("写入 skills 状态文件：%w", err)
	}
	return &Result{Version: st.Version, Skills: st.Skills, Scope: st.Scope}, nil
}

// DriftInfo reports "local skills synced to Current but the CLI is Target".
type DriftInfo struct {
	Current string // synced skills version
	Target  string // running CLI version
}

// Drift returns non-nil only when skills were synced before AND the synced
// version differs from cliVersion. Dev/empty versions never report drift.
func Drift(cliVersion string) *DriftInfo {
	if cliVersion == "" || cliVersion == "dev" {
		return nil
	}
	st := ReadState()
	if st == nil {
		return nil
	}
	if version.Compare(st.Version, cliVersion) == 0 {
		return nil
	}
	return &DriftInfo{Current: st.Version, Target: cliVersion}
}
