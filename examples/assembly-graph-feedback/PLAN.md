# Bound consumer feedback experiment

Before implementation, 2026-10-09. Compiler: public v0.6.22-dev source
d3b44fc63a340d825108c97f61eeb18900219b28, Go 1.27.2.

Extend saved assembly feedback from one activity to a root record assembler
followed by a bound, fixed consumer and a fixed pure helper. Keep the original
root tuple and source-local expectations. Caller expectations describe the final
consumer. Never infer root inputs from intermediate values or invent an oracle.

Predeclare checks:

- Fixed and own-model generation, native saved replay, then bounded construction.
- Omitted model in the new construction makes zero fresh predictions even if the
  origin used a model. Explicit model calls counted per fresh round.
- Final caller field score 6/6 on two separately supplied holdout inputs, with
  zero new inference; training exposure unknown.
- Exact integer 9007199254740993, negative input, false, empty text and Unicode.
- Original source, caller cases and artifact bytes copied unchanged.
- Reject extra roots, dependent assembly inputs, a second assembler, unknown
  entry, absent counters and unsupported prepared assembly helpers.
- Run focused TDD, full workbench race/vet and the original public PR CI.

No model training, new dependencies, compiler changes or time-based result cache.

Follow-up before implementation: the compiler's actual plan uses input_from for
single-argument activities and inputs[].from for multi-argument activities.
Add native single-root coverage and reject an extra single-argument root; retain
the first successful CI observations and check the new commit separately.
