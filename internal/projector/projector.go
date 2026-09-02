package projector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func project(request Request, authority Authority) Decision {
	decision := Decision{
		Status:    StatusClosed,
		Operation: request.Operation,
		Tool:      request.Tool,
		Target:    request.Target,
		Reason:    "accepted within the caller-owned capability boundary",
	}

	capabilities, knownTool := authority.Tools[request.Tool]
	if !knownTool {
		decision.addUnknown("authority", "resolve-tool", "the requested tool has no declared capabilities", "absent_ownership", "obtain a declared tool capability set", "semantic authority")
	}

	if request.AuthorityIdentity == "" {
		decision.addUnknown("authority", "check-identity", "authority identity is missing", "missing_identity", "supply the explicit authority identity", "caller")
	} else if request.AuthorityIdentity != authority.Identity {
		decision.addUnknown("authority", "check-identity", "authority identity does not match the generated authority", "ambiguous_identity", "resolve the authority identity before retrying", "caller")
	}

	if request.AuthorityDigest == "" {
		decision.addUnknown("authority", "check-digest", "authority digest is missing", "missing_digest", "supply the generated authority digest", "caller")
	} else if request.AuthorityDigest != authority.Digest {
		decision.addUnknown("authority", "check-digest", "authority digest is stale or mismatched", "stale_digest", "refresh the generated authority receipt", "caller")
	}

	callerRoot, callerErr := canonicalize(request.CallerOwnedRoot)
	repositoryRoot, repositoryErr := canonicalize(request.RepositoryRoot)
	target, targetErr := canonicalize(request.Target)
	if callerErr != nil || repositoryErr != nil || targetErr != nil {
		decision.addUnknown("canonicalization", "resolve-paths", "one or more authority paths could not be canonicalized", "unresolvable_path", "provide existing roots and a resolvable target", "filesystem")
	} else {
		decision.CanonicalTarget = target
		if request.Operation == OperationCleanup && (samePath(target, callerRoot) || isAncestor(target, callerRoot)) {
			decision.addRefuted("cleanup target is equal to or above the caller-owned root")
		}
		if isWithinOrEqual(target, repositoryRoot) {
			decision.addRefuted("target is inside the repository root; repository writes remain forbidden")
		}
		if hasParentTraversal(request.Target) {
			decision.addRefuted("parent traversal is forbidden before canonical path resolution")
		}

		expanded := expandCapabilities(capabilities, callerRoot, repositoryRoot)
		if !operationWithinDeclaredRoot(request.Operation, target, expanded) {
			decision.addRefuted("target is outside the operation's declared owned root")
		}
		if hasSymlinkEscape(request.Target, target, callerRoot, repositoryRoot) {
			decision.addRefuted("target escapes the caller-owned root through a symlink")
		}
		if request.Operation == OperationCleanup && targetOutsideStrictRoot(target, callerRoot) {
			decision.addRefuted("cleanup target is not strictly below the caller-owned root")
		}
		if request.Operation == OperationAppendEvidence && isExistingEvidence(request, target, expanded.AppendOnlyEvidencePaths) {
			decision.addRefuted("append-only evidence target already exists and cannot be overwritten")
		}
		if overlapsPeerOwnedSubtree(target, request.PeerOwnedRoots, callerRoot, repositoryRoot) {
			decision.addUnknown("ownership", "compare-owned-subtrees", "target overlaps another tool's owned subtree", "ambiguous_ownership", "obtain a non-overlapping ownership receipt", "peer tool authority")
		}
		if containsForbiddenSibling(target, expanded.ForbiddenSiblings) {
			decision.addRefuted("target is a forbidden sibling of the caller-owned root")
		}
	}

	if decision.Status == StatusClosed && decision.CanonicalTarget == "" {
		decision.addUnknown("receipt", "bind-canonical-target", "an accepted operation lacks a canonical target", "missing_canonical_target", "resolve the target before retrying", "canonicalizer")
	}
	decision.ReceiptDigest = digestDecision(decision)
	return decision
}

func (d *Decision) addRefuted(reason string) {
	if StatusRefuted.Rank() >= d.Status.Rank() {
		d.Status = StatusRefuted
		d.Reason = reason
		d.Unknown = nil
	}
}

func (d *Decision) addUnknown(stage, step, reason, unknownClass, nextOperation, blockedBy string) {
	if StatusUnknown.Rank() > d.Status.Rank() {
		d.Status = StatusUnknown
		d.Reason = reason
		d.Unknown = &Unknown{
			Stage:         stage,
			Step:          step,
			Reason:        reason,
			UnknownClass:  unknownClass,
			NextOperation: nextOperation,
			BlockedBy:     blockedBy,
		}
	}
}

func expandCapabilities(capabilities Capabilities, callerRoot, repositoryRoot string) Capabilities {
	return Capabilities{
		InputRoots:              expandList(capabilities.InputRoots, callerRoot, repositoryRoot),
		OwnedOutputRoots:        expandList(capabilities.OwnedOutputRoots, callerRoot, repositoryRoot),
		CleanupRoots:            expandList(capabilities.CleanupRoots, callerRoot, repositoryRoot),
		AppendOnlyEvidencePaths: expandList(capabilities.AppendOnlyEvidencePaths, callerRoot, repositoryRoot),
		ForbiddenAncestors:      expandList(capabilities.ForbiddenAncestors, callerRoot, repositoryRoot),
		ForbiddenSiblings:       expandList(capabilities.ForbiddenSiblings, callerRoot, repositoryRoot),
	}
}

func expandList(values []string, callerRoot, repositoryRoot string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ReplaceAll(value, "${CALLER_ROOT}", callerRoot)
		value = strings.ReplaceAll(value, "${REPOSITORY_ROOT}", repositoryRoot)
		result = append(result, value)
	}
	return result
}

func operationWithinDeclaredRoot(operation Operation, target string, capabilities Capabilities) bool {
	switch operation {
	case OperationProject, OperationWrite, OperationReplay:
		return anyStrictlyWithin(target, capabilities.OwnedOutputRoots)
	case OperationCleanup:
		return anyStrictlyWithin(target, capabilities.CleanupRoots)
	case OperationAppendEvidence:
		for _, evidencePath := range capabilities.AppendOnlyEvidencePaths {
			if samePath(target, evidencePath) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func isExistingEvidence(request Request, target string, evidencePaths []string) bool {
	allowed := false
	for _, evidencePath := range evidencePaths {
		if samePath(target, evidencePath) {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	if request.ExistingTarget != nil {
		return *request.ExistingTarget
	}
	_, err := os.Lstat(target)
	return err == nil
}

func overlapsPeerOwnedSubtree(target string, peerRoots []string, callerRoot, repositoryRoot string) bool {
	for _, rawRoot := range peerRoots {
		root, err := canonicalize(expandList([]string{rawRoot}, callerRoot, repositoryRoot)[0])
		if err == nil && (isWithinOrEqual(root, target) || isWithinOrEqual(target, root)) {
			return true
		}
	}
	return false
}

func containsForbiddenSibling(target string, forbidden []string) bool {
	for _, sibling := range forbidden {
		canonicalSibling, err := canonicalize(sibling)
		if err == nil && (samePath(target, canonicalSibling) || isWithinOrEqual(canonicalSibling, target)) {
			return true
		}
	}
	return false
}

func targetOutsideStrictRoot(target, root string) bool {
	return !isStrictlyWithin(target, root)
}

func anyStrictlyWithin(target string, roots []string) bool {
	for _, root := range roots {
		canonicalRoot, err := canonicalize(root)
		if err == nil && isStrictlyWithin(target, canonicalRoot) {
			return true
		}
	}
	return false
}

func isStrictlyWithin(path, root string) bool {
	return isWithinOrEqual(root, path) && !samePath(path, root)
}

func isWithinOrEqual(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
}

func isAncestor(ancestor, path string) bool {
	return isStrictlyWithin(path, ancestor)
}

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func hasParentTraversal(path string) bool {
	for _, part := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == ".." {
			return true
		}
	}
	return false
}

func hasSymlinkEscape(rawTarget, canonicalTarget, callerRoot, repositoryRoot string) bool {
	if hasParentTraversal(rawTarget) {
		return false
	}
	abs, err := filepath.Abs(rawTarget)
	if err != nil {
		return true
	}
	clean := filepath.Clean(abs)
	if isWithinOrEqual(callerRoot, clean) && !isWithinOrEqual(callerRoot, canonicalTarget) {
		return true
	}
	if isWithinOrEqual(repositoryRoot, clean) && !isWithinOrEqual(repositoryRoot, canonicalTarget) {
		return true
	}
	return false
}

func canonicalize(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("empty path")
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		return filepath.Clean(resolved), nil
	}
	missing := clean
	suffix := make([]string, 0, 4)
	for {
		if _, err := os.Lstat(missing); err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(missing)
			if resolveErr != nil {
				return "", resolveErr
			}
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved), nil
		}
		parent := filepath.Dir(missing)
		if parent == missing {
			return "", fmt.Errorf("no existing ancestor for %q", raw)
		}
		suffix = append(suffix, filepath.Base(missing))
		missing = parent
	}
}

func digestDecision(decision Decision) string {
	encoded, err := decision.MarshalForDigest()
	if err != nil {
		return "sha256:error"
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func ValidateReceiptOutput(outputPath, callerRoot, repositoryRoot string) Decision {
	return Project(Request{
		Tool:              "projector",
		Operation:         OperationProject,
		Target:            outputPath,
		CallerOwnedRoot:   callerRoot,
		RepositoryRoot:    repositoryRoot,
		AuthorityIdentity: GeneratedAuthorityIdentity,
		AuthorityDigest:   GeneratedAuthorityDigest,
	})
}

func EncodeDecision(decision Decision) ([]byte, error) {
	return json.MarshalIndent(decision, "", "  ")
}
