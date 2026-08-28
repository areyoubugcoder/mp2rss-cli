package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// withTempState points StatePath at a temp file for the test's duration.
func withTempState(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "skills-state.json")
	old := StatePath
	StatePath = func() (string, error) { return path, nil }
	t.Cleanup(func() { StatePath = old })
	return path
}

func TestStateRoundTrip(t *testing.T) {
	withTempState(t)
	if got := ReadState(); got != nil {
		t.Fatalf("ReadState on missing file = %+v, want nil", got)
	}
	st := State{Version: "v1.2.0", Skills: []string{"mp2rss-auth"}, Scope: ScopeGlobal, SyncedAt: "2026-08-28T00:00:00Z"}
	if err := WriteState(st); err != nil {
		t.Fatal(err)
	}
	got := ReadState()
	if got == nil || !reflect.DeepEqual(*got, st) {
		t.Errorf("ReadState = %+v, want %+v", got, st)
	}
}

func TestReadStateCorruptOrEmptyVersion(t *testing.T) {
	path := withTempState(t)
	for _, body := range []string{"{not json", `{"skills":["a"],"scope":"global"}`} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := ReadState(); got != nil {
			t.Errorf("ReadState(%q) = %+v, want nil", body, got)
		}
	}
}

func TestReadStateNormalizesScope(t *testing.T) {
	path := withTempState(t)
	if err := os.WriteFile(path, []byte(`{"version":"v1.0.0","scope":"weird"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := ReadState()
	if got == nil || got.Scope != ScopeProject {
		t.Errorf("scope = %v, want project fallback", got)
	}
}

func TestBuildArgs(t *testing.T) {
	if got := BuildArgs(ScopeProject); !reflect.DeepEqual(got, []string{"-y", "skills", "add", RepoSlug, "-y"}) {
		t.Errorf("project args = %v", got)
	}
	if got := BuildArgs(ScopeGlobal); !reflect.DeepEqual(got, []string{"-y", "skills", "add", RepoSlug, "-y", "-g"}) {
		t.Errorf("global args = %v", got)
	}
}

func TestSyncWritesState(t *testing.T) {
	withTempState(t)
	oldRunner := Runner
	defer func() { Runner = oldRunner }()
	var gotArgs []string
	Runner = func(args []string) (string, error) {
		gotArgs = args
		return "ok", nil
	}

	res, err := Sync(ScopeGlobal, "v1.2.0")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if res.Version != "v1.2.0" || res.Scope != ScopeGlobal || len(res.Skills) != len(Official) {
		t.Errorf("result = %+v", res)
	}
	if len(gotArgs) == 0 || gotArgs[len(gotArgs)-1] != "-g" {
		t.Errorf("runner args = %v, want trailing -g", gotArgs)
	}
	st := ReadState()
	if st == nil || st.Version != "v1.2.0" || st.Scope != ScopeGlobal || st.SyncedAt == "" {
		t.Errorf("state after sync = %+v", st)
	}
}

func TestSyncRunnerFailureDoesNotWriteState(t *testing.T) {
	withTempState(t)
	oldRunner := Runner
	defer func() { Runner = oldRunner }()
	Runner = func(_ []string) (string, error) { return "boom", os.ErrPermission }

	if _, err := Sync(ScopeProject, "v1.2.0"); err == nil {
		t.Fatal("expected error")
	}
	if st := ReadState(); st != nil {
		t.Errorf("state should not be written on failure, got %+v", st)
	}
}

func TestDrift(t *testing.T) {
	withTempState(t)
	// no state → nil
	if d := Drift("v1.2.0"); d != nil {
		t.Errorf("drift without state = %+v", d)
	}
	if err := WriteState(State{Version: "v1.1.0", Scope: ScopeProject}); err != nil {
		t.Fatal(err)
	}
	// dev build → nil
	if d := Drift("dev"); d != nil {
		t.Errorf("drift for dev = %+v", d)
	}
	// same version (v-prefix insensitive) → nil
	if d := Drift("1.1.0"); d != nil {
		t.Errorf("drift for equal version = %+v", d)
	}
	// behind → reported
	d := Drift("v1.2.0")
	if d == nil || d.Current != "v1.1.0" || d.Target != "v1.2.0" {
		t.Errorf("drift = %+v", d)
	}
}
