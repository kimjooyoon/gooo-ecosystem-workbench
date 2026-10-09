# Original native typed-path observations

Producer: clean compiler source `34dc675b1b24102014c15e1b22635c481b6858d6`,
Go 1.27.2, embedded decision runtime 0.2.26. This is an unreleased development
producer; the immutable public 0.6.23 source is `2b17c487`.

`typed-fixed`, `typed-mixed` and their `-replay` files are byte-preserved original
CLI outputs from the compiler's unary arithmetic experiment. The fixed run used
2 program attempts; the mixed run used 15. Both subsequent evaluations matched
3/3 supplied caller outputs, with zero new replay predictions. The mixed initial
record assembly made one own-QAT prediction; the single typed decision declined
the three-decision model input contract and continued deterministically.

`typed-rejection` and `typed-fault` are fresh native observations made with that
same binary and the source/case files in `../caller-typed-paths/`. The first used
4 attempts, including one type-rejected combination with no caller score. The
second used 2 attempts, including one native zero-divisor outcome. Both remain
`PARTIAL_FINITE`; their evaluation inputs repeat construction inputs.

The gzip files have zero timestamp metadata and were compared against original
stdout after decompression. `TYPED-PATH-SHA256SUMS` binds their compressed bytes.
Reading and recounting these fixtures does not execute their programs. Native
compiler replay owns source/selection equivalence. The mixed-prefix unit test is
a synthetic transport test and is not counted as a new native experiment.

Historical initial typed local cases are recounted separately. Their source
attempt cap is absent in this receipt, so `budget_known` remains false. A
program candidate's exported palette carries its own explicit source budget.
All observations describe these finite cases; model training exposure is unknown.
