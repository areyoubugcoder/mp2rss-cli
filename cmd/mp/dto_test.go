package mp

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRemoveResultShape locks the JSON contract for `mp remove -o json`:
// keys are camelCase ({"ok":true,"mpId":42}), matching the API DTOs.
func TestRemoveResultShape(t *testing.T) {
	b, err := json.Marshal(removeResult{OK: true, MpID: 42})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"mpId":42`) {
		t.Errorf("expected camelCase mpId in %s", s)
	}
	if strings.Contains(s, "mp_id") {
		t.Errorf("legacy snake_case mp_id leaked: %s", s)
	}
}

// TestSubscribeResultShape locks the JSON contract for `mp subscribe -o json`.
func TestSubscribeResultShape(t *testing.T) {
	b, err := json.Marshal(subscribeResult{OK: true, ArticleURL: "https://mp.weixin.qq.com/s/x"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"articleUrl":"https://mp.weixin.qq.com/s/x"`) {
		t.Errorf("expected camelCase articleUrl in %s", s)
	}
	if strings.Contains(s, "article_url") {
		t.Errorf("legacy snake_case article_url leaked: %s", s)
	}
}
