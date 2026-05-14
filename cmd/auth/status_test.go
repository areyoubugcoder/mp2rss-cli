package auth

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// TestStatusDTOShape locks the public JSON contract for `auth status -o json`:
//   - all keys are camelCase (aligned with API DTOs)
//   - timestamp fields are unix-millis numbers, not formatted strings
func TestStatusDTOShape(t *testing.T) {
	dto := statusDTO{
		LoggedIn:      true,
		Source:        "config",
		APIURL:        "https://mp2rss.bugcode.dev/api",
		FeedKeyMasked: "abcdef***",
		LastLoginAt:   1_705_000_000_000,
		LastVerifyAt:  1_705_000_001_000,
	}
	b, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}

	wantKeys := []string{
		"loggedIn", "source", "apiUrl", "feedKeyMasked", "lastLoginAt", "lastVerifyAt",
	}
	gotKeys := make([]string, 0, len(got))
	for k := range got {
		gotKeys = append(gotKeys, k)
	}
	sort.Strings(gotKeys)
	sort.Strings(wantKeys)
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Errorf("keys = %v, want %v", gotKeys, wantKeys)
	}

	// Timestamps must marshal as numbers, not strings.
	for _, k := range []string{"lastLoginAt", "lastVerifyAt"} {
		if _, ok := got[k].(float64); !ok {
			t.Errorf("%s should be a number, got %T (%v)", k, got[k], got[k])
		}
	}
}

// TestStatusDTOOmitsEmptyTimestamps ensures `omitempty` zeroes don't leak.
func TestStatusDTOOmitsEmptyTimestamps(t *testing.T) {
	dto := statusDTO{LoggedIn: false, Source: "none", APIURL: "https://x"}
	b, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(b, &got)

	for _, k := range []string{"lastLoginAt", "lastVerifyAt", "feedKeyMasked"} {
		if _, present := got[k]; present {
			t.Errorf("%s should be omitted when empty, got %v", k, got[k])
		}
	}
}
