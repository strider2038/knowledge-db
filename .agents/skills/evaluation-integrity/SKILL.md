---
name: evaluation-integrity
description: Build or review evaluation datasets, reference harnesses, and quality gates for detectors, solvers, or model-backed features. Use to prevent circular ground truth, misleading proxy metrics, and invalid comparisons; not for routine unit-test setup.
---

# Evaluation integrity

Start with the claim the evaluation must support. Specify supported inputs, output
semantics, acceptance criteria and the implementation/configuration being measured.
Do not expand product guarantees to match whatever a benchmark happens to contain.

## Separate the questions

Keep distinct result groups when they answer different questions:

| Group | Question | Interpretation |
| --- | --- | --- |
| Contract conformance | Does the implementation meet its documented behavior? | Can gate the declared contract |
| Broader exposure/stress | What happens on difficult, unsupported or open-world inputs? | Diagnoses gaps; gates only if explicitly adopted |
| Lifecycle/performance | What are startup, warm, steady-state and resource costs? | Compare like workloads and lifecycle phases |

Make eligibility and exclusion reasons visible. Do not quietly drop failed cases
or count out-of-scope inputs as failures of a different contract. Report excluded
counts and retain a path for investigating them. Freeze acceptance criteria before
evaluating a candidate; explain and version any later change.

## Establish an independent oracle

Expected outputs must come from a documented specification, independent annotation,
known synthetic construction, or a separately justified reference. Generating
expected results with the implementation under test creates a circular regression
gate. Existing outputs may be characterization snapshots, but label them as such;
they do not independently establish correctness.

An external tool is a reference, not automatically truth. Pin its version, settings,
models/data and preprocessing; inspect disagreements. Preserve provenance and
annotation uncertainty. Keep calibration conventions in one tested adapter:
coordinate axes, row order, units, offsets, normalization and span indexing.
Never scatter sign changes or offset corrections through test callers to make
individual samples pass.

For small or ambiguous datasets, report uncertainty and limits of representativeness.
When human labels are used, define adjudication for ambiguous cases and review a
sample independently where practical. Check source/redistribution rights before
adding external data; keep private or identifying evidence out of public fixtures.

## Validate the harness before trusting the score

Validate generated reference records against the canonical contract before comparison:
required fields, types, identifiers, bounds, ordering and any domain-specific overlap
rules. Reject missing, duplicate or malformed records with actionable diagnostics.
An empty output must not pass merely because an assertion loop had nothing to inspect.

Exercise the harness with deliberately wrong output, malformed records, omitted
samples and a known good case. For a gate that claims to test a fallback or negative
path, ensure the input actually reaches that path. A green run establishes only
what its assertions observe.

## Tie metrics to the end result

Use component metrics to diagnose, then verify that changes improve or preserve the
user-visible outcome. More detections need not mean better recognition; a closer
match to one baseline need not mean a more correct result.

- Keep precision, recall, coverage and abstentions distinguishable when applicable.
- Show relevant input strata and regressions, not only an aggregate average.
- Independently verify proposed hypotheses before calling them successful results
  when the domain supports a stronger check.
- Treat thresholds and dataset membership as versioned evaluation inputs. Do not
  tune them against the acceptance set and present the resulting score as unbiased.
  Maintain a held-out set or disclose that the set was used for development.
- Keep external comparison scores diagnostic unless the product explicitly adopts
  that reference as its compatibility contract.

## Make comparisons reproducible

Record the candidate revision, dirty-tree state if relevant, dataset/manifest version,
reference version, model/configuration, random seeds where useful, runtime/toolchain
and exact invocation. For performance, distinguish cold start, warm-up and steady
state; report repetitions and variability appropriate to the measurement.

Compare baseline and candidate on the same inputs and harness. If the harness or
oracle changed, rerun both or clearly identify the comparison as non-equivalent.
Verify documented defaults in a clean environment so locally fetched artifacts do
not become hidden prerequisites.

## Acceptance output

Provide a compact report containing:

- The claim, evaluated scope, exclusions and oracle provenance.
- Baseline/candidate identifiers and reproducible commands.
- Per-group results, meaningful regressions and failure examples.
- Evidence that the harness rejects incorrect or incomplete results.
- The gate decision and what the evaluation does not establish.

Keep concrete thresholds, dataset paths and product guarantees in the consumer's
specification. This skill supplies the evaluation method, not those policy choices.
