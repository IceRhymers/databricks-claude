package anthropic

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRequest_RoundtripPreservesUnknownFields(t *testing.T) {
	in := []byte(`{
  "model": "claude-opus-4-7",
  "messages": [{"role":"user","content":"hi"}],
  "max_tokens": 1024,
  "temperature": 0.5,
  "metadata": {"user_id": "u123"},
  "stream": true
}`)
	var r Request
	if err := json.Unmarshal(in, &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if r.Model != "claude-opus-4-7" {
		t.Errorf("Model=%q", r.Model)
	}
	if len(r.Messages) != 1 {
		t.Errorf("Messages len=%d", len(r.Messages))
	}
	out, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Canonicalise both for comparison.
	var ain, aout map[string]any
	json.Unmarshal(in, &ain)
	json.Unmarshal(out, &aout)
	canIn, _ := json.Marshal(ain)
	canOut, _ := json.Marshal(aout)
	if !bytes.Equal(canIn, canOut) {
		t.Errorf("roundtrip lost data:\n in=%s\nout=%s", canIn, canOut)
	}
}

func TestIsWebSearchTool(t *testing.T) {
	cases := map[string]bool{
		`{"type":"web_search_20250305","name":"web_search"}`: true,
		`{"type":"web_search_99999999","name":"web_search"}`: true,
		`{"type":"web_fetch_20250910","name":"web_fetch"}`:   false,
		`{"type":"custom","name":"x"}`:                       false,
		`{}`:                                                 false,
	}
	for in, want := range cases {
		if got := IsWebSearchTool([]byte(in)); got != want {
			t.Errorf("IsWebSearchTool(%s)=%v, want %v", in, got, want)
		}
	}
}

func TestIsWebFetchTool(t *testing.T) {
	if !IsWebFetchTool([]byte(`{"type":"web_fetch_20250910"}`)) {
		t.Error("expected true")
	}
	if IsWebFetchTool([]byte(`{"type":"web_search_20250305"}`)) {
		t.Error("expected false")
	}
}

func TestRewriteWebSearchTool_Annotated(t *testing.T) {
	b := RewriteWebSearchTool()
	desc := ToolDescription(b)
	if !IsAnnotatedTool(desc) {
		t.Errorf("rewritten tool description not annotated: %q", desc)
	}
	if ToolName(b) != "web_search" {
		t.Errorf("ToolName=%q", ToolName(b))
	}
	if !strings.Contains(string(b), "query") {
		t.Errorf("rewritten tool missing query schema: %s", b)
	}
}

func TestRewriteWebFetchTool_Annotated(t *testing.T) {
	b := RewriteWebFetchTool()
	if ToolName(b) != "web_fetch" {
		t.Errorf("ToolName=%q", ToolName(b))
	}
	if !IsAnnotatedTool(ToolDescription(b)) {
		t.Errorf("not annotated")
	}
}

// TestAnnotationPrefix_IsWireStable pins AnnotationPrefix to its exact value.
//
// This is NOT a log prefix. It is a wire value: RewriteWebSearchTool and
// RewriteWebFetchTool embed it in the model-visible "description" field of the
// tools they inject, and IsAnnotatedTool matches it back on the response path
// to tell our injected tools apart from user-supplied tools that happen to be
// named web_search / web_fetch. Changing it is a behavior change that breaks
// that round-trip, even though the string contains a tool name.
//
// TestRewriteWebSearchTool_* cover the round trip via
// strings.HasPrefix(desc, AnnotationPrefix), which compares the const against
// itself — so they catch a rename of the const alone (the literals at the
// RewriteWeb*Tool sites would then disagree), but NOT a consistent rename of
// the const and those literals together. This test closes that gap.
func TestAnnotationPrefix_IsWireStable(t *testing.T) {
	const want = "[databricks-claude:websearch] "
	if AnnotationPrefix != want {
		t.Errorf("AnnotationPrefix = %q, want %q\n"+
			"This is a WIRE value round-tripped by IsAnnotatedTool, not a log prefix. "+
			"Changing it breaks identification of proxy-injected tools.", AnnotationPrefix, want)
	}
}
