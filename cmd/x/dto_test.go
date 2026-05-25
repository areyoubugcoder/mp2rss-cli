package x

import (
	"testing"

	"github.com/areyoubugcoder/mp2rss-cli/internal/client"
)

// TestFilterXOnly verifies the client-side fallback filter for x list.
func TestFilterXOnly(t *testing.T) {
	in := []client.Subscription{
		{SourceType: "mp", MpID: 1, MpName: "mp1"},
		{SourceType: "x", XUserID: "44"},
		// server didn't populate sourceType — accept anything with XUserID set.
		{XUserID: "55", XUsername: "u55"},
	}
	out := filterXOnly(in)
	if len(out) != 2 {
		t.Fatalf("want 2 items, got %d (%+v)", len(out), out)
	}
	for _, s := range out {
		if s.XUserID == "" {
			t.Errorf("mp item leaked: %+v", s)
		}
	}
}
