package conformance

import (
	"os"
	"reflect"
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
		if result.ProofChoice == "" || result.Indicator == "" || result.Decision.ProofChoice != result.ProofChoice || result.Decision.Indicator != result.Indicator {
			t.Fatalf("case %d is missing explicit proof/indicator binding", index+1)
		}
	}
	if !reflect.DeepEqual(report.ProofChoiceCounts, map[string]int{"FOUNDATION": 4, "COHERENCE": 4, "REGRESSION": 4}) {
		t.Fatalf("unexpected proof_choice_counts: %#v", report.ProofChoiceCounts)
	}
	if !reflect.DeepEqual(report.IndicatorCounts, map[string]int{"DRIVER": 4, "OUTCOME": 4, "GUARDRAIL": 4}) {
		t.Fatalf("unexpected indicator_counts: %#v", report.IndicatorCounts)
	}
	if report.Cases[11].ReplayMatch == nil || !*report.Cases[11].ReplayMatch {
		t.Fatal("deterministic replay did not match")
	}
	if report.Replay.CellID != "REGRESSION-04" || report.Replay.ProofChoice != "REGRESSION" || report.Replay.Indicator != "GUARDRAIL" || !report.Replay.Match || report.Replay.ReceiptDigest == "" {
		t.Fatalf("incomplete replay artifact: %#v", report.Replay)
	}
	if report.Metrics.RequestedPaths != 12 || report.Metrics.AcceptedOperations != 3 || report.Metrics.UnknownOperations != 3 || report.Metrics.RefutedOperations != 6 {
		t.Fatalf("unexpected operation metrics: %+v", report.Metrics)
	}
	if report.Metrics.RepositoryWrites != 0 || report.Metrics.DestructiveOperations != 0 || report.Metrics.GeneratedArtifactCount != 1 {
		t.Fatalf("unsafe or incomplete metrics: %+v", report.Metrics)
	}
}
