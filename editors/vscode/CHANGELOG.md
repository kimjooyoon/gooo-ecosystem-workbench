# Changelog

## 0.2.2

- Reconnect the language server when `gooo.compilerPath` changes, including after a failed start.
- Serialize client changes and retain the latest compiler path during startup or shutdown.

## 0.2.1

- Highlight `source_fill` and `candidate` as Gooo keywords.
- Use the correct `gooo.compilerPath` setting in the first-run instructions.
