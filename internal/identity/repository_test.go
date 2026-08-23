package identity

import "testing"

func TestNormalizeRepositoryName(t *testing.T) {
	cases := map[string]string{
		" Sergii\\Specview.git ": "sergii/specview",
		"sergii//specview/":       "sergii/specview",
		"SPECVIEW":                "specview",
	}
	for input, want := range cases {
		if got := NormalizeRepositoryName(input); got != want {
			t.Fatalf("NormalizeRepositoryName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeGitRemoteCommonForms(t *testing.T) {
	want := "github.com/sergii/specview"
	for _, remote := range []string{
		"git@github.com:sergii/specview.git",
		"https://github.com/sergii/specview.git",
		"ssh://git@github.com/sergii/specview.git",
	} {
		if got := NormalizeGitRemote(remote); got != want {
			t.Fatalf("NormalizeGitRemote(%q) = %q, want %q", remote, got, want)
		}
	}
	if got := NormalizeGitRemote("../specview"); got != "" {
		t.Fatalf("local filesystem remote must not become cross-host identity: %q", got)
	}
}

func TestRepositoryInstanceIDIsHostLocalAndDeterministic(t *testing.T) {
	hostID := "host:550e8400-e29b-41d4-a716-446655440000"
	first, err := RepositoryInstanceID(hostID, "/home/sergii/repos/specview")
	if err != nil {
		t.Fatal(err)
	}
	second, err := RepositoryInstanceID(hostID, "/home/sergii/repos/./specview")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("canonical roots must yield same instance id: %q != %q", first, second)
	}
	otherHost, err := RepositoryInstanceID("host:550e8400-e29b-41d4-a716-446655440001", "/home/sergii/repos/specview")
	if err != nil {
		t.Fatal(err)
	}
	if otherHost == first {
		t.Fatal("different hosts must not share RepositoryInstance identity")
	}
	otherRoot, err := RepositoryInstanceID(hostID, "/home/sergii/repos/specview-copy")
	if err != nil {
		t.Fatal(err)
	}
	if otherRoot == first {
		t.Fatal("different roots must not share RepositoryInstance identity")
	}
}

func TestRepositoryCorrelationContract(t *testing.T) {
	cases := []struct {
		name  string
		left  RepositoryFingerprint
		right RepositoryFingerprint
		want  CorrelationOutcome
	}{
		{
			name:  "same explicit identity",
			left:  RepositoryFingerprint{ExplicitID: "specview:sergii/specview", Name: "sergii/specview"},
			right: RepositoryFingerprint{ExplicitID: "specview:sergii/specview", Name: "local/specview"},
			want:  CorrelationMatch,
		},
		{
			name:  "different explicit identities",
			left:  RepositoryFingerprint{ExplicitID: "specview:a", Name: "sergii/specview"},
			right: RepositoryFingerprint{ExplicitID: "specview:b", Name: "sergii/specview"},
			want:  CorrelationDistinct,
		},
		{
			name: "same explicit identity with contradictory forge",
			left: RepositoryFingerprint{
				ExplicitID:      "specview:sergii/specview",
				Name:            "sergii/specview",
				ForgeProvider:   "github",
				ForgeRepository: "sergii/specview",
			},
			right: RepositoryFingerprint{
				ExplicitID:      "specview:sergii/specview",
				Name:            "sergii/specview",
				ForgeProvider:   "github",
				ForgeRepository: "other/specview",
			},
			want: CorrelationConflict,
		},
		{
			name: "name and Git remote corroborate",
			left: RepositoryFingerprint{Name: "sergii/specview", GitRemote: "git@github.com:sergii/specview.git"},
			right: RepositoryFingerprint{Name: "SERGII/specview.git", GitRemote: "https://github.com/sergii/specview.git"},
			want: CorrelationMatch,
		},
		{
			name:  "name only is ambiguous",
			left:  RepositoryFingerprint{Name: "sergii/specview"},
			right: RepositoryFingerprint{Name: "sergii/specview"},
			want:  CorrelationAmbiguous,
		},
		{
			name: "same name contradictory Git remote",
			left: RepositoryFingerprint{Name: "sergii/specview", GitRemote: "git@github.com:sergii/specview.git"},
			right: RepositoryFingerprint{Name: "sergii/specview", GitRemote: "git@github.com:other/specview.git"},
			want: CorrelationDistinct,
		},
		{
			name:  "different names without shared explicit identity",
			left:  RepositoryFingerprint{Name: "sergii/specview", GitRemote: "git@github.com:sergii/specview.git"},
			right: RepositoryFingerprint{Name: "local/specview", GitRemote: "git@github.com:sergii/specview.git"},
			want:  CorrelationDistinct,
		},
		{
			name: "forge corroborates name",
			left: RepositoryFingerprint{Name: "sergii/specview", ForgeProvider: "GitHub", ForgeRepository: "sergii/specview"},
			right: RepositoryFingerprint{Name: "sergii/specview", ForgeProvider: "github", ForgeRepository: "SERGII/specview.git"},
			want: CorrelationMatch,
		},
		{
			name: "one explicit identity can correlate with matching ordinary evidence",
			left: RepositoryFingerprint{ExplicitID: "specview:sergii/specview", Name: "sergii/specview", GitRemote: "git@github.com:sergii/specview.git"},
			right: RepositoryFingerprint{Name: "sergii/specview", GitRemote: "https://github.com/sergii/specview.git"},
			want: CorrelationMatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := CorrelateRepositories(tc.left, tc.right)
			if result.Outcome != tc.want {
				t.Fatalf("outcome = %q, want %q; reasons=%#v", result.Outcome, tc.want, result.Reasons)
			}
		})
	}
}
