'use strict';

const { createHash } = require('node:crypto');

function sourceDigest(source) {
  return `sha256:${createHash('sha256').update(source, 'utf8').digest('hex')}`;
}

function parseBodyCodegenResult(stdout, originalSource) {
  const result = JSON.parse(stdout);
  const report = result?.report;
  if (report?.schema !== 'gooo/body-codegen-report/v3' ||
      report.source_digest !== sourceDigest(originalSource) ||
      report.typecheck_passed !== true || report.deterministic_replay !== true ||
      report.repository_writes !== 0 || typeof result.source !== 'string' || result.source.length === 0 ||
      (result.gooo_source !== undefined && (typeof result.gooo_source !== 'string' ||
        result.gooo_source.includes('__GOOO_BODY_HOLE_')))) {
    throw new Error('the compiler returned an incomplete, stale, or unverified body-generation report');
  }
  return result;
}

function canApplyGeneratedSource(generationVersion, currentVersion, expectedDigest, currentSource) {
  return Number.isSafeInteger(generationVersion) && generationVersion === currentVersion &&
    typeof expectedDigest === 'string' && typeof currentSource === 'string' &&
    expectedDigest === sourceDigest(currentSource);
}

function bodyGenerationSummary(result) {
  const report = result.report;
  const lines = [
    `Activity: ${report.activity}`,
    `Generated form: ${result.gooo_source ? 'completed Gooo source' : 'Go projection'}`,
    `Route: ${report.route}`,
    `Body completeness: ${report.completeness_percent}%`,
    `Provider decision latency: ${report.route_decision_latency_ms} ms`
  ];
  if (report.body_fill) {
    const fill = report.body_fill;
    lines.push(
      `Selected assignment: ${fill.selected_candidate_id}`,
      `Training cases: ${fill.test_cases_passed}/${fill.test_cases_total} (${fill.functional_accuracy_percent}%)`,
      `Holdout cases: ${fill.holdout_cases_total === 0
        ? 'not declared'
        : `${fill.holdout_cases_passed}/${fill.holdout_cases_total} (${fill.holdout_accuracy_percent ?? 'unknown'}%)`}`,
      `Provider calls: ${fill.external_provider_calls_known ? fill.external_provider_calls : 'unknown'}`,
      `Provider decision time: ${fill.timing?.provider_decision_ms ?? 0} ms`
    );
  }
  if (report.body_search) {
    const search = report.body_search;
    lines.push(
      `Selected candidate: ${search.selected_candidate_id}`,
      `Training cases: ${search.training_passed}/${search.training_total} (${search.training_accuracy_percent ?? 'unknown'}%)`,
      `Holdout cases: ${search.holdout_total === 0
        ? 'not declared'
        : `${search.holdout_passed}/${search.holdout_total} (${search.holdout_accuracy_percent ?? 'unknown'}%)`}`,
      `Provider decision time: ${search.decision_latency_ms} ms`
    );
  }
  return lines;
}

module.exports = { bodyGenerationSummary, canApplyGeneratedSource, parseBodyCodegenResult, sourceDigest };
