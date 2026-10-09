# Typed graph caller feedback — original local observations

2026-10-10 KST. Workbench producer `f397b52bee518d486a49871c40d2e7e99a1c278f`
was built from clean source with Go 1.27.2. The compiler was the clean, unreleased
unary experiment producer `34dc675b1b24102014c15e1b22635c481b6858d6`.
Its binary SHA256 was `61684ba511aed99852cc35f31f73c2f23e8aeb258003cb66be7e6b45a54ab136`.
These producer identities remain distinct from commits that add these records.

The original mixed graph assembly was produced by installed workbench06445a4
with compiler34dc675b and the existing own-QAT model. It matched 0/1 caller
outputs and made one record prediction. The old Gooo policy stopped at
`expand-joint-profile`. The original graph files were retained and replayed by
the new workbench; the current Gooo policy selected the unchanged caller failure
for bounded native construction.

| Fresh follow-up | Rounds | Program attempts including repeated work | Selected local cases | Consumed caller cases | Final other inputs | Fresh final predictions |
| --- | ---: | ---: | --- | --- | --- | ---: |
| Model omitted | 5 | 25 | 2/2 | 1/1 | 3/3 | 0 |
| Same own-QAT model explicitly requested | 5 | 26 | 2/2 | 1/1 | 3/3 | 0 |

The model-omitted follow-up made zero predictions. The requested-model follow-up
made five initial record predictions, one per fresh round. A single typed choice
does not fit this model's three-decision contract, so its candidates continue
deterministically. These runs do not show a model attempt-count improvement.
They do show that the model-origin graph can be reused in either mode without
changing original source or expected outputs.

The final inputs were 7, -5, and exact integer9007199254740993; construction
consumed input3. Training exposure is unknown. All figures describe this small
declared candidate space and supplied expectations.

The outer CLI wall times were 17.31s and 15.60s; its resource tool reported peak
resident bytes87,998,464 and88,375,296. These are whole CLI observations, not model
inference latency or host CPU utilization. Original time output is retained.

Portable v7 tests recount fixed, mixed, replayed, rejected and native-fault
observations; the original red test exposed unsupported v7 decoding. Focused
race tests completed in6.138s, native candidate integration9.520s, and public
0.6.23 compatibility7.052s. The public compiler's precise unsupported-contract
diagnostic remains tested. The full native regression run is recorded separately
when it terminates; this local record does not claim a completed public PR CI.

Gzip files were byte-compared against original stdout/logs. Each fresh native
round's original JSON is retained. `FILES.sha256` binds payload bytes. Recounting
the saved JSON does not itself rerun its native program.
