'use strict';

const assert = require('node:assert/strict');
const { createHash } = require('node:crypto');
const test = require('node:test');
const { bodyGenerationSummary, canApplyGeneratedSource, parseBodyCodegenResult, sourceDigest } = require('../body-generation');

const originalSource = 'activity Lift(Integer) -> Integer computes "return input"';
const originalDigest = `sha256:${createHash('sha256').update(originalSource, 'utf8').digest('hex')}`;

function report(overrides = {}) {
  return {
    report: {
      schema: 'gooo/body-codegen-report/v3',
      activity: 'Lift',
      source_digest: originalDigest,
      generated_digest: 'sha256:generated',
      replay_digest: 'sha256:replay',
      route: 'preserve',
      route_decision_latency_ms: 0,
      completeness_percent: 100,
      typecheck_passed: true,
      deterministic_replay: true,
      repository_writes: 0,
      ...overrides
    },
    source: 'func Lift(input int64) int64 { return input }',
    gooo_source: 'activity Lift(Integer) -> Integer computes "return input"'
  };
}

test('accepts a source-bound, typechecked, replayed body-codegen result', () => {
  const result = parseBodyCodegenResult(JSON.stringify(report()), originalSource);
  assert.equal(result.gooo_source.includes('__GOOO_BODY_HOLE_'), false);
  assert.equal(result.report.activity, 'Lift');
});

test('rejects malformed, stale, unverified, or writing codegen results', () => {
  assert.throws(() => parseBodyCodegenResult('{', originalSource), SyntaxError);
  assert.throws(() => parseBodyCodegenResult(JSON.stringify(report()), `${originalSource}\n`), /incomplete, stale, or unverified/);
  assert.throws(() => parseBodyCodegenResult(JSON.stringify(report({ typecheck_passed: false })), originalSource), /incomplete, stale, or unverified/);
  assert.throws(() => parseBodyCodegenResult(JSON.stringify(report({ repository_writes: 1 })), originalSource), /incomplete, stale, or unverified/);
});

test('allows application only while the editor document is unchanged', () => {
  assert.equal(canApplyGeneratedSource(8, 8, originalDigest, originalSource), true);
  assert.equal(canApplyGeneratedSource(8, 9, originalDigest, originalSource), false);
  assert.equal(canApplyGeneratedSource(8, 8, originalDigest, `${originalSource}\n`), false);
  assert.equal(canApplyGeneratedSource(8, 8, originalDigest, undefined), false);
  assert.equal(canApplyGeneratedSource(undefined, undefined, originalDigest, originalSource), false);
  assert.equal(sourceDigest(originalSource), originalDigest);
});

test('summarizes observed training, holdout, and provider latency separately', () => {
  const value = report();
  value.report.body_fill = {
    selected_candidate_id: 'compose',
    test_cases_passed: 7,
    test_cases_total: 8,
    functional_accuracy_percent: 87.5,
    holdout_cases_passed: 1,
    holdout_cases_total: 2,
    holdout_accuracy_percent: 50,
    external_provider_calls: 1,
    external_provider_calls_known: true,
    timing: { provider_decision_ms: 425 }
  };
  const summary = bodyGenerationSummary(value).join('\n');
  assert.match(summary, /Training cases: 7\/8 \(87\.5%\)/);
  assert.match(summary, /Holdout cases: 1\/2 \(50%\)/);
  assert.match(summary, /Provider decision time: 425 ms/);
});
