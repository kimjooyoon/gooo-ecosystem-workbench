# Frozen dependent source editing experiment

Freeze before inspecting the existing graph model on this new program.
Use the QAT all-data model published at workbench main 96eeafbb, unchanged.
It trained on filenames, division and retry, and has never trained on Splice.
This is one new program with dependent choices, not hundreds of independent tasks.

The Gooo recipe selects three sequential field updates: whether to apply an
exact-context edit, the new source conditional on that decision, and its byte
length from the updated source. Existing text input/output bounds are 1,024 UTF-8
bytes. Source selection uses the eight authored value_case tuples in the recipe.

Evaluate deterministic order and the unchanged trained graph model at source
attempt budgets 1, 2, 4 and 8. Preserve all proposals, rankings, attempted masks,
case/field completeness, actual native outputs and saved replay. Never label a
partial construction as a complete tool. Model prediction counts must remain
one per modeled construction, zero during native execution and saved replay.

Native inputs use the Cartesian product:
- before: empty, pre:, 한글:, line followed by newline, quoted prefix
- expected fragment: x, 값, emoji 🙂, empty
- after: empty, :tail, :끝, newline
- replacement: empty, replacement, 교체
For each tuple run both matching source and a source with an extra leading !.
Also run replacement-equals-expected no-ops; deduplicate exact five-part tuples.
Add output boundary cases of exactly 1,024 and 1,025 bytes. Assert no tuple occurs
among source selection cases. The independent Go oracle uses prefix/suffix
checks and extracts the intervening original fragment before replacement.
Report the actual roster count; variants share one program and fixed wordings.

Dogfood: use the compiled Gooo Splice to change a small Gooo activity from
`input + 1` to `input * 2`, compile and execute its returned source on negative,
zero and positive integers with separate expectations. Retain stale-context,
no-op and output-limit behavior. No model retraining or post-result plan changes.
