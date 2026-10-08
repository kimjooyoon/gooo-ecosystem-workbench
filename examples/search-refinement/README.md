# Refining a source-owned expression search

Gooo declares the integer hole, expression grammar, examples and attempt budget.
`workbench refine` reads that plan without constructing candidates, then asks
the Gooo next-step program how to proceed after native execution. The existing
Gooo policy can add failed direct inputs to the source cases and increase the
selected activity's budget within the caller's limit.

```sh
go run ./cmd/workbench refine --compiler /path/to/gooo \
  --source examples/search-refinement/source.gooo --activity Add \
  --cases examples/search-refinement/feedback-cases.json \
  --evaluation-cases examples/search-refinement/evaluation-cases.json \
  --policy examples/source-refinement/policy.gooo \
  --max-attempts 8 --max-rounds 4 --out out/search-refined
```

Use compiler `661ec03c5145a252564df2919a109f4a5d8834d8` or a descendant.
The first source declares `2 -> 3` with one attempt. Feedback adds `10 -> 11`.
Evaluation uses `-3 -> -2` and `0 -> 1` only after selecting the program.
These are finite examples of adding one, with no claim about all integer inputs.

## Combine expression search with our small model

`mixed.gooo` binds the generated integer to `Describe`, whose three choices
assemble an Integer/Boolean/String record. Use `mixed-feedback-cases.json`,
`mixed-evaluation-cases.json`, and `--model builtin` with the command above.
Integer search uses its fixed order. The model ranks the three record choices.
All activities still run through compilation, finite checks and native execution.
The source revision policy and next-step program use zero model predictions.

The caller's `--max-attempts` bounds the selected `Add` assembly; other activities
keep their own declared budgets (`Describe` declares eight). The two adaptive
feedback inputs check both activities. Evaluation is kept separate from policy
decisions, but model training exposure is unknown.

## Five reproducible runs

```sh
go build -trimpath -o /tmp/gooo-search-study ./experiments/search-refinement
/tmp/gooo-search-study --compiler /path/to/gooo --out out/search-study
```

The Go driver runs integer search, mixed deterministic construction, mixed model
construction, a grammar missing the needed inner expression, and a two-candidate
cap. Every run saves the original source, source plan, native results, Gooo
decisions, source revisions and separately executed evaluation. Build metadata
in `summary.json` records whether the compiler and experiment used clean sources.

The missing-expression arm embeds the hole in `input + hole`. Its declared
offset grammar derives candidates from final outputs and cannot produce the
needed inner constant `1` from these training cases. The capped arm additionally
omits generated expressions. Gooo distinguishes `expand-declared-choices` from
`expand-search-space`; these proposals remain visible when refinement stops.
Changing the grammar or candidate cap is a next source change, not an automatic
operation performed by this command.

`ranked` in a search observation means retained candidates in the current list.
`omitted` and `space_known` describe the named grammar's generated set. Source
attempt limits, tried candidates, training matches and native outputs stay
separate. Legacy receipts without a budget keep that value unknown.
