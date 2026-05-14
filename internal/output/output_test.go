package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestTruncate_CJKAware(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"abcdefgh", 5, "abcd…"},
		{"中文测试", 4, "中…"},     // 中 is 2 cols → 2 + 1 ellipsis = 3 ≤ 4-1=3 wait
		{"短", 10, "短"},        // fits, no truncation
		{"hello", 5, "hello"}, // exactly fits
		{"", 5, ""},
	}
	for _, c := range cases {
		got := Truncate(c.in, c.max)
		if runewidth.StringWidth(got) > c.max {
			t.Errorf("Truncate(%q, %d) = %q (width %d) > max %d",
				c.in, c.max, got, runewidth.StringWidth(got), c.max)
		}
	}
}

func TestRender_ContainsHeaderAndRows(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, []Column{
		{Header: "ID", Width: 6},
		{Header: "公众号", Width: 10},
	}, [][]string{
		{"42", "Mp2RSS 官方"},
		{"7", ""},
	})
	s := buf.String()
	if !strings.Contains(s, "ID") || !strings.Contains(s, "公众号") {
		t.Errorf("missing header in:\n%s", s)
	}
	if !strings.Contains(s, "─") {
		t.Errorf("missing divider in:\n%s", s)
	}
	if !strings.Contains(s, EmptyPlaceholder) {
		t.Errorf("empty cell not rendered as %q:\n%s", EmptyPlaceholder, s)
	}
}

func TestJSONErrorEnvelopeShape(t *testing.T) {
	var buf bytes.Buffer
	PrintErrorJSON(&buf, "boom", 401)

	var decoded ErrorEnvelope
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, buf.String())
	}
	if decoded.Error.Message != "boom" || decoded.Error.Code != 401 {
		t.Errorf("got %+v, want {boom, 401}", decoded.Error)
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("table"); err != nil {
		t.Error(err)
	}
	if err := Validate("json"); err != nil {
		t.Error(err)
	}
	if err := Validate("yaml"); err == nil {
		t.Error("yaml should be rejected")
	}
}

func TestFormatUnixMillis(t *testing.T) {
	if got := FormatUnixMillis(0); got != EmptyPlaceholder {
		t.Errorf("zero ms should be %q, got %q", EmptyPlaceholder, got)
	}
	// 2024-01-01 00:00:00 UTC = 1704067200000 ms
	got := FormatUnixMillis(1704067200000)
	if got == EmptyPlaceholder || got == "" {
		t.Errorf("non-zero ms must render, got %q", got)
	}
}
