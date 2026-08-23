package hoststate

import (
	"reflect"
	"testing"
)

func TestRepositoryActiveAgentsCollapsesProcessMultiplicity(t *testing.T) {
	repo := Repository{Sessions: []Session{
		{Agent: "Codex", PID: 101, Active: true},
		{Agent: "Claude", PID: 202, Active: true},
		{Agent: "Codex", PID: 303, Active: true},
		{Agent: "Cursor", PID: 404, Active: false},
	}}

	want := []string{"Claude", "Codex"}
	if got := repo.ActiveAgents(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ActiveAgents() = %#v, want %#v", got, want)
	}
}
