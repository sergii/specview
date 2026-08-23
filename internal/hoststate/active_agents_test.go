package hoststate

import (
	"reflect"
	"testing"
)

func TestRepositoryActiveAgentsCollapsesProcessMultiplicity(t *testing.T) {
	repo := Repository{Sessions: []Session{
		{Agent: "Codex", ProcessIDs: []int{101}, Active: true},
		{Agent: "Claude", ProcessIDs: []int{202}, Active: true},
		{Agent: "Codex", ProcessIDs: []int{303}, Active: true},
		{Agent: "Cursor", ProcessIDs: []int{404}, Active: false},
	}}

	want := []string{"Claude", "Codex"}
	if got := repo.ActiveAgents(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ActiveAgents() = %#v, want %#v", got, want)
	}
}
