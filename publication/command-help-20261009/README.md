# Command help observations, 2026-10-09

Base workbench main: ece4da79ee27db30cbf5428ba63896f7d3deb99e.
The new CLI regression first failed on top-level help, per-command help returning
flag.ErrHelp, and an unknown command showing generic flags. The raw failing log
is retained compressed with its original SHA256. The original source/model
observations, expectations and model parameters are unchanged.

After the change, fresh command race tests completed in1.488s and go vet passed.
Three root help forms and both help forms for all12 existing commands return
success; four invalid request forms return errors. A freshly built binary also
returned success for --help, help assemble, construct --help and refine --help.
These local CLI observations do not invoke the Gooo compiler or a model and
are separate from native/model accuracy observations and eventual original CI.

The first screen groups implemented commands by what the user wants to do.
The options come from each existing command parser. Input expectations,
compiler source pins, inference/replay behavior and canonical checks stay the same.
New help files are included in the existing CI formatting step.
