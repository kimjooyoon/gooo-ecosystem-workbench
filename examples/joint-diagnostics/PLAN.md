# Caller construction diagnostics — 2026-10-08

The existing `diagnose` command rejects `body-construct` output. Add a Gooo
recipe that consumes three distinct observations: local obligations in the
selected combination, caller feedback used to construct it, and subsequent
evaluation. Keep whole-program attempts separate from initial local attempts.

Before implementation, retain the complete, partial and model-replayed outputs
from clean compiler 2246ec35710486c4baede7653a24519053917d7b. These are existing
example regressions. They do not measure unseen-task accuracy. The replay
contains historical model ordering, with zero new predictions.

Expected behavior:

- Read actual values and recount local and caller observations for every attempt.
- Preserve exact JSON integers, original input identity, and reported input
  overlap. Do not treat initial local success as whole-program completion.
- Let Gooo decide whether to rerun with a larger program budget, revise declared
  choices/expectations, add evaluation, or retain newly exposed counterexamples.
- A successful evaluation must not hide unsatisfied construction obligations.
- Distinguish input-only evaluation and evaluation containing only consumed roots.
- A diagnostic proposal does not execute a rerun. `body-construct` currently
  restarts bounded construction; saved replay rechecks history and cannot resume
  with a larger budget.
- Verify recipe branches with explicit examples; run native construction both
  deterministically and with the unchanged independently trained graph model.
- Save native observations and generated Gooo diagnostics. Recounting a receipt
  does not itself re-execute its program or verify its source provenance.

During implementation, add a bounded `construct` dispatcher: Gooo computes the
next program budget, and Go invokes the compiler only within the caller's round
and program ceilings. Preserve every fresh run, including repeated work. Fixed
ordering should use budgets 1, 2, 4, 8 on this existing example (15 total program
attempts); the existing graph model should finish in its first one-attempt run.
Other diagnostic actions stop the dispatcher with their reason intact.

The existing six compiler checks and release workflow continue independently.
