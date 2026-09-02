package projector

import "encoding/json"

type Status string

const (
	StatusClosed  Status = "CLOSED"
	StatusUnknown Status = "UNKNOWN"
	StatusRefuted Status = "REFUTED"
)

type Operation string

const (
	OperationProject        Operation = "project"
	OperationWrite          Operation = "write"
	OperationCleanup        Operation = "cleanup"
	OperationAppendEvidence Operation = "append_evidence"
	OperationReplay         Operation = "replay"
)

type Unknown struct {
	Stage         string `json:"stage"`
	Step          string `json:"step"`
	Reason        string `json:"reason"`
	UnknownClass  string `json:"unknown_class"`
	NextOperation string `json:"next_operation"`
	BlockedBy     string `json:"blocked_by"`
}

type Decision struct {
	Status          Status    `json:"status"`
	Operation       Operation `json:"operation"`
	Tool            string    `json:"tool"`
	Target          string    `json:"target"`
	CanonicalTarget string    `json:"canonical_target,omitempty"`
	Reason          string    `json:"reason"`
	Unknown         *Unknown  `json:"unknown,omitempty"`
	ReceiptDigest   string    `json:"receipt_digest"`
}

type Request struct {
	Tool                 string    `json:"tool"`
	Operation            Operation `json:"operation"`
	Target               string    `json:"target"`
	CallerOwnedRoot      string    `json:"caller_owned_root"`
	RepositoryRoot       string    `json:"repository_root"`
	AuthorityIdentity    string    `json:"authority_identity"`
	AuthorityDigest      string    `json:"authority_digest"`
	PeerOwnedRoots       []string  `json:"peer_owned_roots,omitempty"`
	ExistingTarget       *bool     `json:"existing_target,omitempty"`
}

type Capabilities struct {
	InputRoots               []string `json:"input_roots"`
	OwnedOutputRoots         []string `json:"owned_output_roots"`
	CleanupRoots             []string `json:"cleanup_roots"`
	AppendOnlyEvidencePaths  []string `json:"append_only_evidence_paths"`
	ForbiddenAncestors       []string `json:"forbidden_ancestors"`
	ForbiddenSiblings        []string `json:"forbidden_siblings"`
}

type Authority struct {
	Identity             string                 `json:"identity"`
	Digest               string                 `json:"digest"`
	Precedence           []Status               `json:"precedence"`
	Tools                map[string]Capabilities `json:"tools"`
	RepositoryWrites     int                    `json:"repository_writes"`
	DestructiveOperations int                   `json:"destructive_operations_executed"`
}

type Metrics struct {
	RequestedPaths            int    `json:"requested_paths"`
	OwnedRoots                int    `json:"owned_roots"`
	AcceptedOperations        int    `json:"accepted_operations"`
	UnknownOperations         int    `json:"unknown_operations"`
	RefutedOperations         int    `json:"refuted_operations"`
	AncestorDeleteAttempts    int    `json:"ancestor_delete_attempts"`
	SiblingOverlapAttempts    int    `json:"sibling_overlap_attempts"`
	RepositoryWrites          int    `json:"repository_writes"`
	DestructiveOperations     int    `json:"destructive_operations_executed"`
	GeneratedArtifactCount    int    `json:"generated_artifact_count"`
	WallMilliseconds          *int64 `json:"wall_ms"`
	RSSBytes                  *int64 `json:"rss_bytes"`
	MeasurementStatus         Status `json:"measurement_status"`
}

func (s Status) Rank() int {
	switch s {
	case StatusRefuted:
		return 3
	case StatusUnknown:
		return 2
	default:
		return 1
	}
}

func (d Decision) MarshalForDigest() ([]byte, error) {
	copyDecision := d
	copyDecision.ReceiptDigest = ""
	return json.Marshal(copyDecision)
}
