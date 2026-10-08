# First source-graph chooser study

Freeze this plan before training or inspecting chooser results. Corpus authority
is workbench main `145922bf9dfd55f8bb05fb85938a4bb0ff4b9d6d`: three Gooo program
families, eight requested policies, eight arrangements, three authored wordings.
The 576 rows have independent finite labels and explicit source graph contexts.

Use the existing SDK v0.2.26 graph ABI: a shared 256-input / 8-ReLU / 2-output
field judge (2,072 parameters), applied to three fields and composed into eight
mask scores. No model pretraining or external language weights. Training uses
Go with fixed arrays and CPU; this first small experiment does not measure GPU
training. Source features and label extraction stay outside learned parameters.

## Fixed training

The settings in `plan.json` apply to all folds without validation-based tuning:
Adam (beta1 0.9, beta2 0.999, epsilon 1e-8), shuffled mini-batches of 64 fields,
binary cross-entropy on each field's two logits, no weight decay, no early
stopping. Float32 master weights and activations; float64 gradient/optimizer
accumulation. Seed 20261008 initializes every fold independently. Report full
training loss traces, including failures or lack of improvement.

After FP32 training, post-training quantization uses each matrix's mean absolute
weight as scale, round-to-nearest and clamp to {-1,0,1}. Biases stay FP32. A second
copy of the FP32 master weights receives 200 QAT epochs: quantized forward pass,
identity straight-through gradient on weights, and detached per-matrix scales
recomputed for each batch. Export FP32, PTQ and QAT in the existing SDK format.
Five trits per byte use 1.6 stored bits per matrix weight; log2(3) is about 1.585.
Report the full model payload and actual resident tensor bytes separately.

## Evaluation and controls

Hold out one whole family at a time: 384 training rows and 192 evaluation rows.
All its wordings, policies and arrangements stay excluded. The fourth run uses
all 576 rows for a clearly labeled in-sample construction demonstration, with no
held-out accuracy claim. Fixed corpus order determines fold order.

Reload every exported artifact with the public Go SDK. Score first mask, three
field bits, and pairs whose requests differ in exactly one field. Report both
the exact predicted bit flip and whether both predictions match their labels;
a consistently wrong flip alone is not successful intent following. Keep row
predictions and split membership. Repeat inference on the same graph with intent
withheld. Measure the existing v1 model only after explicitly projecting each
graph back to its first/second expression and intent fields using the v1 ABI.

Measure load time, prepared-array inference wall time and tensor sizes separately.
Timing is one local observation without a controlled host-load comparison.

Use trained QAT weights directly in actual Gooo construction and finite checking,
retaining model proposals, final selections, attempt counts, source-case/field
counts and saved replay. Include finite-budget partial results. Native inputs
remain disjoint from source selection and are excluded from training and model
features. Declare the exact subset executed; model row accuracy and native
program correctness are separate measures.

No hyperparameter choice may use the held-out results in this run. If these
programs later guide architecture or feature changes, treat them as development
data and add new program families for a fresh evaluation. This small authored
corpus does not measure arbitrary natural-language programming.
