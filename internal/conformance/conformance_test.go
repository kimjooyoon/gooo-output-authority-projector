package conformance

import (
	"os"
	"testing"

	"github.com/kimjooyoon/gooo-output-authority-projector/internal/projector"
)

func TestTwelveCaseVector(t *testing.T) {
	root := t.TempDir()
	repositoryRoot := root + "/repository"
	if err := os.MkdirAll(repositoryRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := Run(root, repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cases) != 12 {
		t.Fatalf("case count = %d, want 12", len(report.Cases))
	}
	want := []projector.Status{
		projector.StatusClosed,
		projector.StatusClosed,
		projector.StatusRefuted,
		projector.StatusRefuted,
		projector.StatusRefuted,
		projector.StatusRefuted,
		projector.StatusRefuted,
		projector.StatusUnknown,
		projector.StatusUnknown,
		projector.StatusUnknown,
		projector.StatusRefuted,
		projector.StatusClosed,
	}
	for index, result := range report.Cases {
		if result.ExpectedStatus != want[index] || result.Decision.Status != want[index] {
			t.Fatalf("case %d status = %s/%s, want %s", index+1, result.Decision.Status, result.ExpectedStatus, want[index])
		}
		if result.Decision.Status == projector.StatusUnknown {
			unknown := result.Decision.Unknown
			if unknown == nil || unknown.Stage == "" || unknown.Step == "" || unknown.Reason == "" || unknown.UnknownClass == "" || unknown.NextOperation == "" || unknown.BlockedBy == "" {
				t.Fatalf("case %d has incomplete UNKNOWN receipt", index+1)
			}
		}
	}
	if report.Cases[11].ReplayMatch == nil || !*report.Cases[11].ReplayMatch {
		t.Fatal("deterministic replay did not match")
	}
	if report.Metrics.RequestedPaths != 12 || report.Metrics.AcceptedOperations != 3 || report.Metrics.UnknownOperations != 3 || report.Metrics.RefutedOperations != 6 {
		t.Fatalf("unexpected operation metrics: %+v", report.Metrics)
	}
	if report.Metrics.RepositoryWrites != 0 || report.Metrics.DestructiveOperations != 0 || report.Metrics.GeneratedArtifactCount != 1 {
		t.Fatalf("unsafe or incomplete metrics: %+v", report.Metrics)
	}
}
