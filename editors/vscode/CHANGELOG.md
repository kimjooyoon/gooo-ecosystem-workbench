# Changelog

## 0.2.4

- Generate both library starter routes using the source-declared plans provided by the pinned compiler template.
- Open generated Gooo source only after a complete passing receipt with both native executions and all finite cases passing.
- Verify starter output, source bindings, and saved-receipt replay against the public compiler and Go 1.27.2 in CI.

## 0.2.3

- Start the pinned public compiler with `gooo lsp`, without the unsupported `--stdio` argument added by an explicit transport setting.
- Exercise the installed language-client adapter and the public compiler's diagnostics and completion over stdio in CI.

## 0.2.2

- Reconnect the language server when `gooo.compilerPath` changes, including after a failed start.
- Serialize client changes and retain the latest compiler path during startup or shutdown.

## 0.2.1

- Highlight `source_fill` and `candidate` as Gooo keywords.
- Use the correct `gooo.compilerPath` setting in the first-run instructions.
