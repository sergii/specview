package executionhistory

import "testing"

func TestFindRequiresExactRepositoryAndSessionPair(t *testing.T) {
	projection := Projection{Entries: []Entry{
		{RepositoryID: "repo-a", SessionID: "shared", Agent: "Codex"},
		{RepositoryID: "repo-b", SessionID: "shared", Agent: "Claude"},
	}}

	entry, ok := Find(projection, "repo-b", "shared")
	if !ok || entry.Agent != "Claude" {
		t.Fatalf("exact lookup = %#v, %v", entry, ok)
	}
	if _, ok := Find(projection, "repo-c", "shared"); ok {
		t.Fatal("lookup matched session in another repository")
	}
	if _, ok := Find(projection, "repo-a", "missing"); ok {
		t.Fatal("lookup matched missing session")
	}
	if _, ok := Find(projection, "", "shared"); ok {
		t.Fatal("lookup matched without repository id")
	}
}
