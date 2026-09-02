package conformance

import (
	"os"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-output-authority-projector/internal/projector"
)

type Case struct {
	ID           string
	CellID       string
	CellKind     string
	ActivityID   string
	ActivityKind string
	Name         string
	Vector       []string
	Request      projector.Request
	Expected     projector.Status
}

type CaseResult struct {
	ID             string             `json:"id"`
	CellID         string             `json:"cell_id"`
	CellKind       string             `json:"cell_kind"`
	ActivityID     string             `json:"activity_id"`
	ActivityKind   string             `json:"activity_kind"`
	Name           string             `json:"name"`
	Vector         []string           `json:"vector"`
	ExpectedStatus projector.Status   `json:"expected_status"`
	Decision       projector.Decision `json:"decision"`
	ReplayMatch    *bool              `json:"replay_match,omitempty"`
}

type ExternalState struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type Report struct {
	AuthorityIdentity         string             `json:"authority_identity"`
	AuthorityDigest           string             `json:"authority_digest"`
	StatusPrecedence          []projector.Status `json:"status_precedence"`
	Cases                     []CaseResult       `json:"cases"`
	Metrics                   projector.Metrics  `json:"metrics"`
	ExternalUtility           ExternalState      `json:"external_utility"`
	PerformanceImprovement    ExternalState      `json:"performance_improvement"`
	LocalValidationCount      int                `json:"local_validation_count"`
	OperationalRefutedHistory []string           `json:"operational_refuted_history"`
}

func Run(callerRoot, repositoryRoot string) (Report, error) {
	if err := prepareFixtures(callerRoot, repositoryRoot); err != nil {
		return Report{}, err
	}
	authority := projector.GeneratedAuthority()
	cases := buildCases(callerRoot, repositoryRoot)
	results := make([]CaseResult, 0, len(cases))
	accepted, unknown, refuted := 0, 0, 0
	ancestorDeleteAttempts, siblingOverlapAttempts := 0, 0
	for _, testCase := range cases {
		decision := projector.Project(testCase.Request)
		replayMatch := (*bool)(nil)
		if testCase.Name == "deterministic-replay" {
			replay := projector.Project(testCase.Request)
			match := decisionsEqual(decision, replay)
			replayMatch = &match
		}
		switch decision.Status {
		case projector.StatusClosed:
			accepted++
		case projector.StatusUnknown:
			unknown++
		case projector.StatusRefuted:
			refuted++
		}
		if testCase.Name == "shared-temp-ancestor-deletion" {
			ancestorDeleteAttempts++
		}
		if testCase.Name == "sibling-deletion" || testCase.Name == "overlapping-ownership" {
			siblingOverlapAttempts++
		}
		results = append(results, CaseResult{
			ID:             testCase.ID,
			CellID:         testCase.CellID,
			CellKind:       testCase.CellKind,
			ActivityID:     testCase.ActivityID,
			ActivityKind:   testCase.ActivityKind,
			Name:           testCase.Name,
			Vector:         testCase.Vector,
			ExpectedStatus: testCase.Expected,
			Decision:       decision,
			ReplayMatch:    replayMatch,
		})
	}
	return Report{
		AuthorityIdentity: authority.Identity,
		AuthorityDigest:   authority.Digest,
		StatusPrecedence:  authority.Precedence,
		Cases:             results,
		Metrics: projector.Metrics{
			RequestedPaths:         len(cases),
			OwnedRoots:             len(authority.Tools["projector"].OwnedOutputRoots) + len(authority.Tools["evidence-writer"].OwnedOutputRoots),
			AcceptedOperations:     accepted,
			UnknownOperations:      unknown,
			RefutedOperations:      refuted,
			AncestorDeleteAttempts: ancestorDeleteAttempts,
			SiblingOverlapAttempts: siblingOverlapAttempts,
			RepositoryWrites:       authority.RepositoryWrites,
			DestructiveOperations:  authority.DestructiveOperations,
			GeneratedArtifactCount: projector.GeneratedArtifactCount,
			WallMilliseconds:       nil,
			RSSBytes:               nil,
			MeasurementStatus:      projector.StatusUnknown,
		},
		ExternalUtility: ExternalState{
			Status: "UNKNOWN",
			Reason: "no external utility has been measured with an equivalent authority identity",
		},
		PerformanceImprovement: ExternalState{
			Status: "UNKNOWN",
			Reason: "exact same-identity before/after evidence is absent",
		},
		LocalValidationCount:      0,
		OperationalRefutedHistory: []string{},
	}, nil
}

func prepareFixtures(callerRoot, repositoryRoot string) error {
	for _, path := range []string{
		callerRoot,
		filepath.Join(callerRoot, "output"),
		filepath.Join(callerRoot, "output", "tool-cache"),
		filepath.Join(callerRoot, "output", "overlap"),
		filepath.Join(callerRoot, "evidence"),
		filepath.Join(filepath.Dir(callerRoot), "outside"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	link := filepath.Join(callerRoot, "output", "escape-link")
	if _, err := os.Lstat(link); os.IsNotExist(err) {
		if err := os.Symlink(filepath.Join(filepath.Dir(callerRoot), "outside"), link); err != nil {
			return err
		}
	}
	return nil
}

func buildCases(callerRoot, repositoryRoot string) []Case {
	authority := projector.GeneratedAuthority()
	base := func(operation projector.Operation, target string) projector.Request {
		return projector.Request{
			Tool:              "projector",
			Operation:         operation,
			Target:            target,
			CallerOwnedRoot:   callerRoot,
			RepositoryRoot:    repositoryRoot,
			AuthorityIdentity: authority.Identity,
			AuthorityDigest:   authority.Digest,
		}
	}
	appendTarget := filepath.Join(callerRoot, "evidence", "replay.jsonl")
	overlap := filepath.Join(callerRoot, "output", "overlap", "result.json")
	existing := true
	return []Case{
		{ID: "C01", CellID: "FOUNDATION-01", CellKind: "FOUNDATION", ActivityID: "DRIVER-01", ActivityKind: "DRIVER", Name: "valid-nested-output", Vector: []string{"status=CLOSED", "operation=project", "boundary=nested-owned-output"}, Request: base(projector.OperationProject, filepath.Join(callerRoot, "output", "report.json")), Expected: projector.StatusClosed},
		{ID: "C02", CellID: "FOUNDATION-02", CellKind: "FOUNDATION", ActivityID: "DRIVER-02", ActivityKind: "DRIVER", Name: "valid-own-subtree-cleanup", Vector: []string{"status=CLOSED", "operation=cleanup", "boundary=owned-subtree"}, Request: base(projector.OperationCleanup, filepath.Join(callerRoot, "output", "tool-cache", "stale")), Expected: projector.StatusClosed},
		{ID: "C03", CellID: "FOUNDATION-03", CellKind: "FOUNDATION", ActivityID: "DRIVER-03", ActivityKind: "DRIVER", Name: "shared-temp-ancestor-deletion", Vector: []string{"status=REFUTED", "operation=cleanup", "boundary=ancestor"}, Request: base(projector.OperationCleanup, filepath.Dir(callerRoot)), Expected: projector.StatusRefuted},
		{ID: "C04", CellID: "FOUNDATION-04", CellKind: "FOUNDATION", ActivityID: "DRIVER-04", ActivityKind: "DRIVER", Name: "sibling-deletion", Vector: []string{"status=REFUTED", "operation=cleanup", "boundary=sibling"}, Request: base(projector.OperationCleanup, filepath.Join(filepath.Dir(callerRoot), "sibling")), Expected: projector.StatusRefuted},
		{ID: "C05", CellID: "COHERENCE-01", CellKind: "COHERENCE", ActivityID: "OUTCOME-01", ActivityKind: "OUTCOME", Name: "repository-root-write", Vector: []string{"status=REFUTED", "operation=write", "boundary=repository-root"}, Request: base(projector.OperationWrite, repositoryRoot), Expected: projector.StatusRefuted},
		{ID: "C06", CellID: "COHERENCE-02", CellKind: "COHERENCE", ActivityID: "OUTCOME-02", ActivityKind: "OUTCOME", Name: "symlink-escape", Vector: []string{"status=REFUTED", "operation=project", "boundary=symlink"}, Request: base(projector.OperationProject, filepath.Join(callerRoot, "output", "escape-link", "report.json")), Expected: projector.StatusRefuted},
		{ID: "C07", CellID: "COHERENCE-03", CellKind: "COHERENCE", ActivityID: "OUTCOME-03", ActivityKind: "OUTCOME", Name: "parent-traversal", Vector: []string{"status=REFUTED", "operation=project", "boundary=traversal"}, Request: base(projector.OperationProject, callerRoot+string(os.PathSeparator)+"output"+string(os.PathSeparator)+".."+string(os.PathSeparator)+"outside"), Expected: projector.StatusRefuted},
		{ID: "C08", CellID: "COHERENCE-04", CellKind: "COHERENCE", ActivityID: "OUTCOME-04", ActivityKind: "OUTCOME", Name: "missing-authority", Vector: []string{"status=UNKNOWN", "operation=project", "unknown_class=missing_identity"}, Request: projector.Request{Tool: "projector", Operation: projector.OperationProject, Target: filepath.Join(callerRoot, "output", "missing.json"), CallerOwnedRoot: callerRoot, RepositoryRoot: repositoryRoot, AuthorityDigest: authority.Digest}, Expected: projector.StatusUnknown},
		{ID: "C09", CellID: "REGRESSION-01", CellKind: "REGRESSION", ActivityID: "GUARDRAIL-01", ActivityKind: "GUARDRAIL", Name: "stale-authority-digest", Vector: []string{"status=UNKNOWN", "operation=project", "unknown_class=stale_digest"}, Request: projector.Request{Tool: "projector", Operation: projector.OperationProject, Target: filepath.Join(callerRoot, "output", "stale.json"), CallerOwnedRoot: callerRoot, RepositoryRoot: repositoryRoot, AuthorityIdentity: authority.Identity, AuthorityDigest: "sha256:stale"}, Expected: projector.StatusUnknown},
		{ID: "C10", CellID: "REGRESSION-02", CellKind: "REGRESSION", ActivityID: "GUARDRAIL-02", ActivityKind: "GUARDRAIL", Name: "overlapping-ownership", Vector: []string{"status=UNKNOWN", "operation=project", "unknown_class=ambiguous_ownership"}, Request: withPeer(base(projector.OperationProject, overlap), "${CALLER_ROOT}/output/overlap"), Expected: projector.StatusUnknown},
		{ID: "C11", CellID: "REGRESSION-03", CellKind: "REGRESSION", ActivityID: "GUARDRAIL-03", ActivityKind: "GUARDRAIL", Name: "append-only-overwrite", Vector: []string{"status=REFUTED", "operation=append_evidence", "boundary=append-only"}, Request: withExisting(base(projector.OperationAppendEvidence, appendTarget), &existing), Expected: projector.StatusRefuted},
		{ID: "C12", CellID: "REGRESSION-04", CellKind: "REGRESSION", ActivityID: "GUARDRAIL-04", ActivityKind: "GUARDRAIL", Name: "deterministic-replay", Vector: []string{"status=CLOSED", "operation=replay", "replay=identical-receipt"}, Request: base(projector.OperationReplay, filepath.Join(callerRoot, "output", "replay.json")), Expected: projector.StatusClosed},
	}
}

func withPeer(request projector.Request, peer string) projector.Request {
	request.PeerOwnedRoots = []string{peer}
	return request
}

func withExisting(request projector.Request, existing *bool) projector.Request {
	request.ExistingTarget = existing
	return request
}

func decisionsEqual(left, right projector.Decision) bool {
	left.ReceiptDigest = ""
	right.ReceiptDigest = ""
	return reflect.DeepEqual(left, right)
}
