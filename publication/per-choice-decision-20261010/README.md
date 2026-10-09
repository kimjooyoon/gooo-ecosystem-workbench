# Original own per-choice decision study and native use

Observed2026-10-10 KST. Clean trainer
`c2cde2d1be126c758382c3769563eef1f45e86bb`; compiler context/native producer
`426caecb711e47da26fe659237a117f301d112f4`. Go1.27.2/public SDK0.2.26.
The model learns from scratch and emits path scores and possible abstention.

| Variant | Train labels | Other utterance labels | Other abstentions | Packed bytes | Resident tensors |
| --- | --- | --- | --- | --- | --- |
| FP32 |48/48|16/24|0/24|50,912|50,912|
| Post-training ternary |34/48|11/24|9/24|2,759|12,896|
| Quantization-aware ternary |48/48|15/24|5/24|2,759|12,896|

Wrong predictions with confidence≥0.8 were3/2/1 respectively. Scores are
uncalibrated. All24 other utterances share six source snapshots and related
direction wording with48 training rows. This is a paraphrase study; unseen
program families and general language accuracy remain unmeasured. Original
study JSON retains every ID, split, input array and output.

SDK timings cover1,2,3,6,16 independent sites,200 serial repeats with a fixed
worker workspace. FP32 averaged8.7..8.95µs/site and ternary about10µs/site.
QAT16-site mean was159,770.835ns. Compression reduced file storage, while its
inference was slower in this local run. Focused allocation tests observed0
heap objects per call. Host CPU utilization is UNMEASURED; CPU was used and
GPU was not. Both training phases took183,416,750ns; full process timing/RSS
is retained separately.

## Actual Gooo construction

- Unary one-choice preflightREADY_FOR_RANKING,0 predictions/tests and the exact
  input SHA later consumed by construction.
- Model1 call/12,958ns,0 external calls, declared layout_reverse selected.
- One program attempt within limit2; local1/1 and caller1/1.
- Other native inputs3/3, replay3/3 and0 new replay/evaluation calls.
- Exact input9007199254740993/output18014398509481986 checked through Go
  json.Number strings, with model/feature/compiler identities verified.
- Whole construction1.25s/maxRSS86,753,280bytes includes native compilation
  and execution, separately from inference latency.

Raw CLI outputs retain source, cases, selected code and history. Success for
this one program does not erase the wrong high-confidence classification or
establish probability calibration. No default compiler/workbench model changed.

## Validation and preservation

Initial compile failures (SDK workspace method vs function and unused repeat
binding) are retained. Corrected focused race1.637s and vet passed: gradient,
SDK export/load logits, fixed sizes and0-allocation tests. Complete workbench
race and vet passed separately; terminal logs are retained. The first source
export probe used an unsupported --source flag; the corrected positional-source
command produced the six copied inputs. Failed output is not a successful export.

Each gzip-n payload was decompressed and byte-compared. FP32/PTQ weights are
exact copies; QAT is at`models/per-choice-20261010/qat_ternary`. FILES.sha256
binds this archive; MODEL.sha256 binds the QAT artifact. No executable/heavy
model is included. HF auth was unavailable and no HF upload is claimed.
The unpublished token-generation prototype was removed after the user clarified
decision-only scope; it is outside this study.
