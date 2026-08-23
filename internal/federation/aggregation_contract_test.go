package federation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sergii/specview/internal/identity"
)

func TestFederationAggregationSafetyContractFixture(t *testing.T) {
	var fixture struct {
		SchemaVersion int       `json:"schema_version"`
		ObservedAt    time.Time `json:"observed_at"`
		Cases         []struct {
			Name      string `json:"name"`
			Instances []struct {
				HostID             string                         `json:"host_id"`
				Hostname           string                         `json:"hostname"`
				Root               string                         `json:"root"`
				SourceRepositoryID string                         `json:"source_repository_id"`
				Fingerprint        identity.RepositoryFingerprint `json:"fingerprint"`
			} `json:"instances"`
			Expected struct {
				RepositoryGroups  int                         `json:"repository_groups"`
				CorrelationIssues int                         `json:"correlation_issues"`
				Outcome           identity.CorrelationOutcome `json:"outcome"`
				Reason            string                      `json:"reason"`
			} `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(readFederationFixture(t, "v1-aggregation-cases.json"), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != 1 || fixture.ObservedAt.IsZero() {
		t.Fatalf("unexpected aggregation contract metadata: %#v", fixture)
	}

	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			snapshots := make([]HostSnapshot, 0, len(tc.Instances))
			for _, item := range tc.Instances {
				snapshot := snapshotForFingerprint(t, item.HostID, item.Hostname, item.Root, item.SourceRepositoryID, item.Fingerprint)
				snapshot.ObservedAt = fixture.ObservedAt
				snapshots = append(snapshots, snapshot)
			}

			projection, err := NewAggregator().Aggregate(snapshots...)
			if err != nil {
				t.Fatal(err)
			}
			if len(projection.Repositories) != tc.Expected.RepositoryGroups {
				t.Fatalf("repository groups = %d, want %d", len(projection.Repositories), tc.Expected.RepositoryGroups)
			}
			if len(projection.Issues) != tc.Expected.CorrelationIssues {
				t.Fatalf("correlation issues = %d, want %d: %#v", len(projection.Issues), tc.Expected.CorrelationIssues, projection.Issues)
			}
			for _, issue := range projection.Issues {
				if issue.Outcome != tc.Expected.Outcome {
					t.Fatalf("issue outcome = %q, want %q: %#v", issue.Outcome, tc.Expected.Outcome, issue)
				}
				if !containsReason(issue.Reasons, tc.Expected.Reason) {
					t.Fatalf("issue reasons = %#v, want %q", issue.Reasons, tc.Expected.Reason)
				}
			}
		})
	}
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}
