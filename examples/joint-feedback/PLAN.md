# Automatic caller counterexamples — 2026-10-08

Starting state: workbench e1e35fd8 and public Gooo 0.6.12, compiler source
4a0cfdd8. `construct` currently stops at `add-counterexamples-to-construction`.
Preserve expectations and append failing evaluation rows to subsequent caller
construction. Gooo decides row inclusion and the next program budget; Go copies
case bytes, tracks provenance and invokes the compiler.

## Before model use

Freeze this plan and examples before running the existing graph model. Do not
train or edit weights. Compare deterministic order and the unchanged graph QAT
model where the source has three choices. Treat this as a small functional
study, not a population estimate.

- Existing three-choice arithmetic program: initial caller `0 -> 0` accepts
  insufficient choices. Adaptive examples expose positive and negative failures.
- New three-choice charge calculation: preserve ten-times base and two-times
  rebate, apply the rebate at inputs >=100. Alternatives change each component.
  Zero is an insufficient local/caller example. Report model outcomes as observed.
- Contradictory caller/evaluation rows must remain together and finish partial.
- Include records/multiple expected activities, exact integers above 2^53,
  duplicate rows, shuffled trace order and input-only observations in reader tests.
- Original construction/evaluation bytes remain saved unchanged. Every appended
  row has source evaluation index, original row digest, originating result hash,
  previous/new construction-file hashes and Gooo decision evidence.
- Repeated evaluations are adaptive. Final holdout, when supplied, is passed to
  the compiler only after selection stops and is never fed back. Keep input
  overlap, counts and new prediction counts explicit.
- Round/program limits still apply. Compiler case limits (128 rows / 32 KiB)
  produce a retained limit result. Duplicate rows cannot cause endless feedback.
- Replay the final construction with zero new inference. CI exercises the new
  Gooo policy and actual compiler loop, plus existing ecosystem tools.

Missing source choices remain a separate language extension. This work adds
constraints to declared choices; it does not invent an oracle or weaken one.
