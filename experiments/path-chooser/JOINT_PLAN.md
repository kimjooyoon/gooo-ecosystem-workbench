# One frozen joint-target continuation study

Start both arms from the published own FP32 path model, weights SHA256
8d5f63c6e27abd703eb88b13b7487c9ff57d48f6ebe6b68ca119182309482f06.
Keep256→48→8,12,728 parameters, semantic_context_intent_v3 and the existing
source-owned candidate vocabulary. No token generation or external base model.

## Data and fixed arms

- Rehearse the original48 authored label examples. Their24 different utterances
  remain a development regression set; they share the original six source shapes.
- Consume the frozen compatible-path corpus from PR59: the exact three source
  inputs and joint acceptable masks1 and2, derived from nine absolute-value cases.
  Keep source/context/cases/target/native provenance. Those nine cases are training
  evidence and are never counted as new functional evaluation.
- Label-only control:100 additional FP32 epochs then100 QAT epochs, batch8,
  learning rate.002, the existing deterministic shuffle and reset Adam per phase.
- Joint arm: the same schedule. Each batch mixes mean label cross-entropy and
  joint negative log accepted probability mass at equal weight. The joint term
  is evaluated once per batch. Record both loss components and exact update count.
- Export FP32 and QAT for both arms. Record original old QAT as a separate reference.
  No hyperparameter search or retry based on the first observed result.

The forward model still factorizes per-site scores. The joint loss preserves
correlations in the target set, but a factorized distribution can prefer one
working mode rather than representing every working mode. Allowed-pair mass
is a ranking quantity, not calibrated probability of functional correctness.

## Native use after training

Use the installed public0.6.24 compiler and unchanged three-choice Gooo source.
Construction consumes only its original local0→0 and caller3→6 cases. Evaluate
the old QAT, control QAT and joint QAT in that fixed order, then saved replay
for each. Preserve preflight/actual model-input parity and zero new replay calls.
This order permits cache effects, so wall time is not a causal speed comparison.

Freeze new caller inputs before training: -11,-9,-8,-2,2,4,8,9,11 and
±9007199254740995. Expect0 for negative Main inputs and exactly twice a positive
input. These11 tuples do not occur in the nine target cases, the original caller
case, or the previous five feedback evaluations. They still exercise the same
authored function family: no unseen-source or whole-int64 accuracy claim.

Report actual selected masks, local/caller attempts, finite11-case outcomes,
saved replay, new model calls, SDK timing and whole-command resource readings.
Separate source-family generalization and calibrated abstention remain open.
