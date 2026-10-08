# Gooo source-graph chooser: first trained artifacts

Twelve independently trained/exported artifacts: four data splits × FP32, PTQ
and QAT. Shared 256→8→2 field judge, 2,072 parameters, Go CPU training.
Use `triple_record_value_graph_v3_shared_v1` with decision SDK v0.2.26-experimental
and Gooo 0.6.9-dev. Each QAT/PTQ weight file is 446 bytes. Five trits per byte
store matrix weights at 1.6 bits each; FP32 biases and metadata are additional.

`filenames`, `division`, and `retry` name the entire family excluded from training.
`all-data-demonstration` was trained on all 576 authored rows. Its results describe
in-sample behavior. Full frozen settings, evaluation limits, poor family transfer,
native construction and replay are in the [study](../../publication/graph-chooser-20261008/README.md).
These experimental artifacts are selected by an explicit file path; the existing
`--model builtin` continues to use the earlier v1 model.

Example from the repository root:

```sh
gooo body-codegen --json --activity Classify \
  --path-model models/graph-chooser-20261008/filenames/qat_ternary/model.json \
  examples/graph-choice-corpus/filenames.gooo
```

The model ranks source-declared alternatives. Gooo checks finite cases and
retains the selected body and evidence. Model-free ordering and saved replay
remain available. These weights learned only from the repository's authored
Gooo graph/intent corpus; no external model weights were used. MIT license.
Current distribution: GitHub. This graph-model study has no completed HF upload.
