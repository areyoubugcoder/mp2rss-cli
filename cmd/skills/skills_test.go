package skills

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/areyoubugcoder/mp2rss-cli/internal/cliopts"
	"github.com/areyoubugcoder/mp2rss-cli/internal/output"
	"github.com/areyoubugcoder/mp2rss-cli/internal/skills"
	"github.com/areyoubugcoder/mp2rss-cli/internal/version"
)

func testDeps(format string) *cliopts.Deps {
	return &cliopts.Deps{
		Output: func() string { return format },
		APIKey: func() string { return "" },
		APIURL: func() string { return "" },
	}
}

func withTempState(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "skills-state.json")
	old := skills.StatePath
	skills.StatePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { skills.StatePath = old })
}

func TestListJSON(t *testing.T) {
	root := NewCmd(testDeps(output.FormatJSON))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var got []skills.Skill
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("list -o json 不是合法 JSON：%v\n%s", err, buf.String())
	}
	if len(got) != len(skills.Official) {
		t.Errorf("list returned %d skills, want %d", len(got), len(skills.Official))
	}
}

func TestStatusNeverSynced(t *testing.T) {
	withTempState(t)
	root := NewCmd(testDeps(output.FormatTable))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "未通过 mp2rss 同步过") {
		t.Errorf("status output:\n%s", buf.String())
	}
}

func TestSyncUpToDateShortCircuit(t *testing.T) {
	withTempState(t)
	oldVer := version.Version
	version.Version = "v9.9.9"
	t.Cleanup(func() { version.Version = oldVer })
	if err := skills.WriteState(skills.State{
		Version: "v9.9.9", Skills: []string{"mp2rss-auth"}, Scope: skills.ScopeProject, SyncedAt: "2026-08-28T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	// Runner 若被调用直接判失败：up-to-date 时不应起 npx。
	oldRunner := skills.Runner
	skills.Runner = func(_ []string) (string, error) {
		t.Fatal("runner should not be invoked when already synced")
		return "", nil
	}
	t.Cleanup(func() { skills.Runner = oldRunner })

	root := NewCmd(testDeps(output.FormatTable))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"sync"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "已是最新") {
		t.Errorf("sync output:\n%s", buf.String())
	}
}

func TestSyncRunsRunnerAndReports(t *testing.T) {
	withTempState(t)
	oldVer := version.Version
	version.Version = "v9.9.9"
	t.Cleanup(func() { version.Version = oldVer })
	oldRunner := skills.Runner
	skills.Runner = func(_ []string) (string, error) { return "ok", nil }
	t.Cleanup(func() { skills.Runner = oldRunner })

	root := NewCmd(testDeps(output.FormatJSON))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"sync", "--global"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	// JSON 模式下输出必须是纯 JSON（可直接 jq），不允许混进度行。
	var dto struct {
		Synced  bool   `json:"synced"`
		Version string `json:"version"`
		Scope   string `json:"scope"`
	}
	if err := json.Unmarshal(buf.Bytes(), &dto); err != nil {
		t.Fatalf("sync -o json 不是纯 JSON：%v\n%s", err, buf.String())
	}
	if !dto.Synced || dto.Version != "v9.9.9" || dto.Scope != "global" {
		t.Errorf("dto = %+v", dto)
	}
	if st := skills.ReadState(); st == nil || st.Scope != skills.ScopeGlobal {
		t.Errorf("state = %+v", st)
	}
}
