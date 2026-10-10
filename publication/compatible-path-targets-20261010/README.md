# Function-compatible joint targets on this PC

The [Go observer](../../experiments/path-outcomes) enumerated the eight declared
paths of the original three-choice Gooo `Choose` activity. Producer
`81bdc74bcb578371d59334f2f52c664dd1251c19` used the installed public
Gooo0.6.24/source366 and decision SDK0.2.26, with Go1.27.2.
The compiler exported a fresh source-bound plan and inputs without inference.

All nine **consumed** activity examples request absolute value. These are target
construction data, informed by the earlier caller failure, not held-out cases.
Of eight combinations, masks1 and2 satisfy all nine. Their branch and comparison
directions differ while both keep the subtraction order. If their per-site
labels are treated independently, four combinations become possible and two
fail. Training targets therefore retain the full compatible mask set.

Native execution of each unchanged SDK Go projection agreed with its interpreter
for all72 results, including ±9007199254740993 without floating conversion.
This checks projection parity, not72/72 intended functionality: only two complete
candidates satisfy the target examples. The finite output groups were{0,3},
{1,2}, and{4,5,6,7}; no claim covers unobserved inputs or the full caller Main.

The one original run used0.96s wall,0.18s user+0.20s system CPU time, and
74,924,032B maximum resident set size reported by the timed command, which
includes compiler export and native Go compilation/execution. This is not a
sum of simultaneous process memory. The in-process
SDK enumeration took187,459ns. Host CPU utilization was not measured; GPU and
model inference were unused. There was no weight update or accuracy comparison.

Original source, consumed cases, context, exact candidate bodies, interpreter
outcomes, native source/output and producer metadata are gzip-n archives under
`original/`. Every archive was decompressed and compared byte for byte before
publication; `FILES.sha256` covers this bundle. `summary.json` selects fields
from the complete original `targets.json.gz` without recomputing the run.

The original focused tests passed. An additional test initially used an
unsupported `divide` operation and failed during plan preparation; its log is
retained. The corrected test checks that a reported native fault cannot be
confused with a zero result. The final focused race suite and vet passed.
Other tests cover changed source/plan/input rejection, native missing rows,
single-bit large-integer disagreement, duplicate cases, cancellation, empty
targets and the exhaustive64-combination limit.

Next training work can optimize probability mass over compatible **joint**
selections. Per-site marginal counts are diagnostics, not independent correct
labels. Families must be split before adding intent paraphrases. Existing
model weights and the earlier2/3 caller result remain unchanged.
