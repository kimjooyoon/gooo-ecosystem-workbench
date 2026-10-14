# Validation and editor releases

Native validation runs for pull requests, main pushes, and non-editor tags.
Open a pull request to validate a branch change, or use the workflow's manual
run for a diagnostic check. The editor package workflow keeps its existing
source-path filters for pull requests and main pushes. Manual runs remain
available, but do not replace main push evidence for publication.

`vscode-v*` tags use the editor release workflow. Generic native and editor
package workflows exclude those duplicate tag events. All compiler identity,
Go/Gooo execution, model, fixture, and editor integration checks remain in their
original jobs; there is no source-path exclusion in native validation.

Immediately before publication, `check-release-validation.py` requires:

- The checked-out commit and official tag target to equal `GITHUB_SHA`.
- The repository identity to match the workflow context, and the tag commit to
  remain an ancestor of current main.
- Successful main **push** runs for `ci.yml` and `vscode-extension.yml` at that
  exact commit. A PR, manual run, tag run, or different commit is insufficient.
- The latest activity and current attempt for each workflow, including a rerun
  of an older run ID; an earlier successful attempt cannot hide a newer failure.
- The expected jobs and validation steps to complete successfully. Missing,
  skipped, failed, incomplete or oversized API evidence refuses publication.

The gate can wait up to five minutes for identified live validation jobs. It
rechecks the tag, ancestry, selected attempts and job/step outcomes before
returning success. The report is `out/release-validation.json`; the check does
not publish anything itself and is not atomic with later GitHub changes.

Choose a tag commit whose native and editor package **main push** checks have
already run. A docs-only commit can have no package run because of its path
filters; missing evidence is a refusal rather than permission to reuse an older
commit's result. Each tag uses the workflow revision stored at that tag.

Run the standard-library validator tests locally with:

```sh
python3 .github/scripts/test_check_release_validation.py
```

The recorded fixture contains selected fields from real GitHub 0.2.4 metadata.
Mutated records and mocked CLI cases test validator behavior only. CI also reads
its current workflow/run/attempt jobs with the actual Actions token to verify
API access and identity; that observation is not a release decision.
