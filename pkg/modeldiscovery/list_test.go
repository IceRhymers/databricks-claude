package modeldiscovery

import (
	"reflect"
	"testing"
)

func TestListAnthropicModels(t *testing.T) {
	cases := []struct {
		name     string
		services []Service
		want     []Model
	}{
		{
			name:     "empty input",
			services: nil,
			want:     []Model{},
		},
		{
			name: "mixed input skips ambiguous and non-messages",
			services: []Service{
				// classifies: opus 4.7 (1M)
				svc("workspace.default.opus", messages, "system.ai.databricks-claude-opus-4-7"),
				// ambiguous: dests span two families -> skipped
				svc("workspace.default.mixed", messages,
					"system.ai.databricks-claude-opus-4-7",
					"system.ai.databricks-claude-sonnet-4-6"),
				// unparseable dest -> skipped
				svc("workspace.default.weird", messages, "system.ai.some-other-model"),
				// no destinations -> skipped
				svc("workspace.default.nodest", messages),
				// does not support messages -> skipped
				svc("workspace.default.nomsg", []string{"other/v1"},
					"system.ai.databricks-claude-haiku-4-5"),
				// classifies: haiku 4.5 (never 1M)
				svc("workspace.default.haiku", messages, "system.ai.databricks-claude-haiku-4-5"),
			},
			want: []Model{
				{FQN: "workspace.default.opus", OneM: true},
				{FQN: "workspace.default.haiku", OneM: false},
			},
		},
		{
			name: "1M boundary",
			services: []Service{
				// sonnet 4.6 qualifies for 1M
				svc("workspace.default.s46", messages, "system.ai.databricks-claude-sonnet-4-6"),
				// sonnet 4.5 does not qualify
				svc("workspace.default.s45", messages, "system.ai.databricks-claude-sonnet-4-5"),
				// opus 5.0 qualifies
				svc("workspace.default.o50", messages, "system.ai.databricks-claude-opus-5-0"),
				// haiku 4.9 never qualifies
				svc("workspace.default.h49", messages, "system.ai.databricks-claude-haiku-4-9"),
			},
			want: []Model{
				{FQN: "workspace.default.o50", OneM: true},
				{FQN: "workspace.default.h49", OneM: false},
				{FQN: "workspace.default.s46", OneM: true},
				{FQN: "workspace.default.s45", OneM: false},
			},
		},
		{
			name: "sort order newest first with FQN tie-break",
			services: []Service{
				svc("workspace.default.b", messages, "system.ai.databricks-claude-opus-4-6"),
				svc("workspace.default.a", messages, "system.ai.databricks-claude-opus-4-6"),
				svc("workspace.default.newer", messages, "system.ai.databricks-claude-opus-4-7"),
			},
			want: []Model{
				{FQN: "workspace.default.newer", OneM: true},
				// same version 4.6 -> stable FQN ascending tie-break
				{FQN: "workspace.default.a", OneM: true},
				{FQN: "workspace.default.b", OneM: true},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ListAnthropicModels(c.services)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ListAnthropicModels() = %+v, want %+v", got, c.want)
			}
		})
	}
}
