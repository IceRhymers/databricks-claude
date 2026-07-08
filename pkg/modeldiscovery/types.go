// Package modeldiscovery auto-discovers the newest Claude model per family
// (opus/sonnet/haiku) that supports the Anthropic Messages API from Unity
// Catalog model-services. It is designed to work even under narrow Unity
// Catalog grants, where the metastore-scoped LIST endpoint is filtered by the
// caller's EXECUTE permissions.
//
// The package is pure stdlib and has no external dependencies. The resolution
// logic (Resolve) is a pure function with no I/O so it can be exhaustively
// unit-tested; the HTTP-backed functions (ListServices, GetService, Discover)
// wrap it with Unity Catalog API access.
package modeldiscovery

// Service is a Unity Catalog model-service securable as observed via the
// model-services API.
type Service struct {
	// FQN is the service securable fully-qualified name, e.g.
	// "workspace.default.acme-claude-opus" or "system.ai.claude-opus-4-8".
	FQN string
	// Catalog is the first dot-segment of FQN, e.g. "workspace" or "system".
	Catalog string
	// SupportedAPITypes lists the API types the service exposes, e.g.
	// ["anthropic/v1/messages"]. It may be empty in the LIST response for
	// user-deployed services and is then enriched via GetService.
	SupportedAPITypes []string
	// Destinations holds the parsed routing.destinations[] entries.
	Destinations []Dest
}

// Dest is a single parsed routing destination of a Service.
type Dest struct {
	// FQN is the routing destination name, e.g.
	// "system.ai.databricks-claude-opus-4-8".
	FQN string
	// Family is the Claude family: "opus", "sonnet", "haiku", or "" when the
	// destination FQN does not match the expected shape.
	Family string
	// Major is the parsed major version; zero when Parsed is false.
	Major int
	// Minor is the parsed minor version; zero when Parsed is false.
	Minor int
	// Parsed is false when FQN did not match the
	// claude-(opus|sonnet|haiku)-major-minor shape.
	Parsed bool
}

// Pins holds caller-supplied verbatim FQN overrides per family. A non-empty pin
// short-circuits discovery for that family.
type Pins struct {
	Opus   string
	Sonnet string
	Haiku  string
}

// Resolved is the outcome of resolving a single family to a service FQN.
type Resolved struct {
	// FQN is the emitted service FQN. When OneM is true the literal "[1m]"
	// suffix has already been appended.
	FQN string
	// OneM reports whether the service qualifies for the 1M-context variant.
	OneM bool
}

// ModelSet is the resolved model per Claude family.
type ModelSet struct {
	Opus   Resolved
	Sonnet Resolved
	Haiku  Resolved
}

// Unresolved describes a family that could not be resolved to any service,
// along with a copy-pasteable remediation hint.
type Unresolved struct {
	// Family is the unresolved Claude family: "opus", "sonnet", or "haiku".
	Family string
	// PinCommand is a copy-pasteable remediation hint, e.g. a pin command.
	PinCommand string
}
