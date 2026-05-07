// Package anthropic models just enough of Anthropic's Messages API to
// safely rewrite tools[] and intercept tool_result blocks for the
// --with-websearch workaround. Unknown fields pass through unmolested.
package anthropic

import (
	"bytes"
	"encoding/json"
	"strings"
)

// AnnotationPrefix tags the Description of a tool we injected on the request
// path, so we recognise it on the response/next-request path.
const AnnotationPrefix = "[databricks-claude:websearch] "

// ClientToolName_WebSearch is the canonical client-tool name we rewrite
// web_search_* server tools to.
const ClientToolName_WebSearch = "web_search"

// ClientToolName_WebFetch is the canonical client-tool name for web_fetch_*.
const ClientToolName_WebFetch = "web_fetch"

// Request is the minimal-but-roundtrip-safe shape of an Anthropic
// /v1/messages request body. All fields not explicitly named end up in
// Extras and are re-serialised verbatim, preserving forward-compatibility
// with future Anthropic API additions.
type Request struct {
	Model    string            `json:"-"`
	Messages []json.RawMessage `json:"-"`
	Tools    []json.RawMessage `json:"-"`
	System   json.RawMessage   `json:"-"`
	Extras   map[string]json.RawMessage
}

// UnmarshalJSON implements custom JSON decoding that captures unknown fields
// in Extras so MarshalJSON can re-emit them.
func (r *Request) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Extras = map[string]json.RawMessage{}
	for k, v := range raw {
		switch k {
		case "model":
			_ = json.Unmarshal(v, &r.Model)
		case "messages":
			_ = json.Unmarshal(v, &r.Messages)
		case "tools":
			_ = json.Unmarshal(v, &r.Tools)
		case "system":
			r.System = v
		default:
			r.Extras[k] = v
		}
	}
	return nil
}

// MarshalJSON re-emits the canonical fields plus any preserved Extras.
func (r *Request) MarshalJSON() ([]byte, error) {
	out := map[string]json.RawMessage{}
	for k, v := range r.Extras {
		out[k] = v
	}
	if r.Model != "" {
		b, err := json.Marshal(r.Model)
		if err != nil {
			return nil, err
		}
		out["model"] = b
	}
	if r.Messages != nil {
		b, err := json.Marshal(r.Messages)
		if err != nil {
			return nil, err
		}
		out["messages"] = b
	}
	if r.Tools != nil {
		b, err := json.Marshal(r.Tools)
		if err != nil {
			return nil, err
		}
		out["tools"] = b
	}
	if len(r.System) > 0 {
		out["system"] = r.System
	}
	return json.Marshal(out)
}

// IsWebSearchTool returns true if the given tool JSON object's "type" field
// has the prefix "web_search_".
func IsWebSearchTool(toolJSON []byte) bool { return hasTypePrefix(toolJSON, "web_search_") }

// IsWebFetchTool returns true if the type field has the prefix "web_fetch_".
func IsWebFetchTool(toolJSON []byte) bool { return hasTypePrefix(toolJSON, "web_fetch_") }

func hasTypePrefix(toolJSON []byte, prefix string) bool {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(toolJSON, &probe); err != nil {
		return false
	}
	return strings.HasPrefix(probe.Type, prefix)
}

// IsAnnotatedTool returns true if a tool's description starts with our
// annotation prefix, indicating we (the proxy) injected it.
func IsAnnotatedTool(description string) bool {
	return strings.HasPrefix(description, AnnotationPrefix)
}

// RewriteWebSearchTool returns the canonical client-tool replacement for any
// web_search_* server tool.
func RewriteWebSearchTool() []byte {
	const tmpl = `{
  "name": "web_search",
  "description": "[databricks-claude:websearch] Search the web. Returns a JSON list of {title, url, snippet} results. Use sparingly; prefer specific queries.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {"type": "string", "description": "search query"}
    },
    "required": ["query"]
  }
}`
	return compactJSON([]byte(tmpl))
}

// RewriteWebFetchTool returns the canonical client-tool replacement for any
// web_fetch_* server tool.
func RewriteWebFetchTool() []byte {
	const tmpl = `{
  "name": "web_fetch",
  "description": "[databricks-claude:websearch] Fetch a URL and return its readable text content. Honours robots.txt; capped at a configurable byte budget.",
  "input_schema": {
    "type": "object",
    "properties": {
      "url": {"type": "string", "description": "absolute http(s) URL to fetch"}
    },
    "required": ["url"]
  }
}`
	return compactJSON([]byte(tmpl))
}

func compactJSON(b []byte) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		return b
	}
	return buf.Bytes()
}

// ToolName extracts the "name" field from a tool JSON object.
func ToolName(toolJSON []byte) string {
	var probe struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(toolJSON, &probe)
	return probe.Name
}

// ToolDescription extracts the "description" field from a tool JSON object.
func ToolDescription(toolJSON []byte) string {
	var probe struct {
		Description string `json:"description"`
	}
	_ = json.Unmarshal(toolJSON, &probe)
	return probe.Description
}
