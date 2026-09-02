# Gooo output authority projector

This repository turns the repeated “cleaned the whole Actions temp directory” failure mode into a semantic authority rule. The `.gooo` authority declares, per tool, input roots, owned output roots, cleanup roots, append-only evidence paths, forbidden ancestors, and forbidden siblings. The generated Go projector canonicalizes paths and emits a bounded capability receipt without executing cleanup.

## Contract

The contract contains exactly 12 cells and exactly 12 Gooo activities: four FOUNDATION, four COHERENCE, and four REGRESSION cells; plus four DRIVER, four OUTCOME, and four GUARDRAIL activities. Each cell explicitly emits one `proof_choice` and one `indicator`; their exact counts are four per declared value. Each cell exposes a vector, never a scalar score. Status precedence is `REFUTED > UNKNOWN > CLOSED`.

The deterministic vector covers valid nested output, valid own-subtree cleanup, shared-temp ancestor deletion, sibling deletion, repository-root write, symlink escape, parent traversal, missing authority, stale authority digest, overlapping ownership, append-only overwrite, and deterministic replay. A known boundary violation is REFUTED. Absent, stale, or ambiguous ownership is UNKNOWN, with stage, step, reason, unknown class, next operation, and blocked-by evidence in every UNKNOWN receipt.

Runtime writes are restricted to a caller-owned output subtree. Repository writes and destructive operations are both fixed at zero. Conformance fixtures only project and verify decisions; they never invoke cleanup. The generated artifact count is one. Wall time and RSS are reported as `null` with `UNKNOWN` measurement status because no exact same-identity measurement is claimed. External utility and performance improvement remain UNKNOWN until equivalent identity evidence exists.

## Use

Project a JSON request from standard input:

```json
{
  "tool": "projector",
  "operation": "project",
  "target": "/tmp/caller-owned/output/report.json",
  "caller_owned_root": "/tmp/caller-owned",
  "repository_root": "/workspace/repository",
  "authority_identity": "gooo-output-authority-projector@0.1",
  "authority_digest": "sha256:<generated digest>"
}
```

```sh
go run ./cmd/gooo-output-authority-projector project < request.json
```

The command only writes a receipt when `--output` itself is accepted as a nested caller-owned output path. The `cleanup` operation is a projection decision; it does not remove anything.

GitHub Actions is the validation boundary. It uses Go 1.27.x to format-check, test, vet, build, and run safe conformance, then uploads one validation artifact. Local validation count for the release record is zero.

## Release policy

Release tags are annotated and never rewritten. The release workflow refuses an existing release, creates a draft first, attaches digests, and publishes only after draft creation. v0.1.0 is preserved exactly as a semantic-refuted predecessor because its artifact lacked literal proof/indicator fields; v0.1.1 carries the correction. Any historical non-immutable release would be preserved as `OPERATIONAL_REFUTED` and followed by the next immutable version.
