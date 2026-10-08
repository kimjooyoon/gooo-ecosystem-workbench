# Included Gooo shared-field model

Unchanged public QAT weights from
[asketeddy/gooo-record-shared-field-tiny-v1](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1/tree/5800946afb35506d66357ee3ea6f956f506795b9/models/qat_ternary),
MIT license. 2,072 parameters; 446-byte weight file; contract
`triple_record_field_context_v1_shared_v1`.

Weight SHA256: `049081cc6dc62f3c4230f72e601c1fd421c514fce9cfdf7bc1eb412ca4c5514b`.
Metadata SHA256: `f95925a277c45957722bc116213c61bee997cc07a94adae738a05d218d660059`.

The model ranks the eight combinations of three source-declared choices.
Gooo checks their finite cases and compiles the chosen body. Omit `--model` for
deterministic order. These v1 weights remain unchanged. The newer source-graph
contract has a [separate Go training study](../graph-chooser-20261008/README.md)
with explicit family splits and independently published artifacts.

The [candidate-order audit](../../examples/feature-audit/README.md) found that
the current feature projection maps eight filename candidate arrangements to
two arrays. The unchanged model accepts the supplied choice in 1/8 first
predictions; a deterministic chooser with this exact input is limited to 2/8
for those labels. The source examples still reach their finite expected results
through Gooo candidate checks. This positional result qualifies earlier
single-order observations in which the model needed one candidate.
