'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const textmate = require('vscode-textmate');
const oniguruma = require('vscode-oniguruma');

const grammarPath = process.env.GOOO_GRAMMAR_PATH || path.join(__dirname, '../syntaxes/gooo.tmLanguage.json');
const source = fs.readFileSync(grammarPath, 'utf8');
const wasm = fs.readFileSync(require.resolve('vscode-oniguruma/release/onig.wasm'));
const onigLib = oniguruma.loadWASM(wasm.buffer.slice(wasm.byteOffset, wasm.byteOffset + wasm.byteLength)).then(() => ({
  createOnigScanner(patterns) { return new oniguruma.OnigScanner(patterns); },
  createOnigString(value) { return new oniguruma.OnigString(value); }
}));

function loadGrammar(raw = source) {
  const registry = new textmate.Registry({
    onigLib,
    loadGrammar(scope) { return scope === 'source.gooo' ? textmate.parseRawGrammar(raw, grammarPath) : null; }
  });
  return registry.loadGrammar('source.gooo').then(grammar => ({ grammar, registry }));
}

function tokenize(grammar, lines) {
  let state = textmate.INITIAL;
  return lines.map(line => {
    const result = grammar.tokenizeLine(line, state);
    state = result.ruleStack;
    return result.tokens.map(token => ({ text: line.slice(token.startIndex, token.endIndex), start: token.startIndex, end: token.endIndex, scopes: token.scopes }));
  });
}

function tokenAt(tokens, position) {
  const token = tokens.find(value => value.start <= position && value.end > position);
  assert.ok(token, `no actual TextMate token at ${position}`);
  return token;
}

function assertScope(tokens, position, scope) {
  assert.ok(tokenAt(tokens, position).scopes.includes(scope), `expected ${scope} at ${position}: ${JSON.stringify(tokens)}`);
}

test('real Gooo source-fill assembly produces both keyword scopes', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const lines = fs.readFileSync(path.resolve(__dirname, '../../../examples/caller-source-fill/source.gooo'), 'utf8').split(/\r?\n/);
    const actual = tokenize(grammar, lines);
    const fill = lines.findIndex(line => /^\s*source_fill\b/.test(line));
    const candidates = lines.map((line, index) => ({ line, index })).filter(row => /^\s*candidate\b/.test(row.line));
    assert.ok(fill >= 0, 'real fixture must contain source_fill');
    assert.equal(candidates.length, 3);
    assertScope(actual[fill], lines[fill].indexOf('source_fill'), 'keyword.other.gooo');
    for (const row of candidates) {
      assertScope(actual[row.index], row.line.indexOf('candidate'), 'keyword.other.gooo');
      assertScope(actual[row.index], row.line.indexOf('"') + 1, 'string.quoted.double.gooo');
    }
    for (const word of ['package', 'entity', 'activity', 'assembling']) {
      const index = lines.findIndex(line => new RegExp(`\\b${word}\\b`).test(line));
      assert.ok(index >= 0);
      const scope = word === 'assembling' ? 'keyword.other.gooo' : 'keyword.declaration.package.gooo';
      assertScope(actual[index], lines[index].indexOf(word), scope);
    }
  } finally { registry.dispose(); }
});

test('engine-authored condition cases receive the assembly keyword scope', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    // meta-ontology-go@8d057a805d284653cf68c5dcf4efc8cc415d1d5a,
    // examples/body-codegen/source-condition-cases.gooo.fixture:16-18.
    const lines = [
      '    condition_case "comparison" input "-9007199254740995" -> "true"',
      '    condition_case "comparison" input "0" -> "false"',
      '    condition_case "comparison" input "9007199254740995" -> "false"'
    ];
    const actual = tokenize(grammar, lines);
    for (let index = 0; index < lines.length; index++) {
      const position = lines[index].indexOf('condition_case');
      assertScope(actual[index], position, 'keyword.other.gooo');
      assert.equal(tokenAt(actual[index], position).text, 'condition_case');
      assertScope(actual[index], lines[index].indexOf('comparison'), 'string.quoted.double.gooo');
    }
  } finally { registry.dispose(); }
});

test('condition_case stays text inside quoted strings and comments', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const lines = [
      'intent "condition_case"',
      'computes `condition_case`',
      '// condition_case "comparison" input "0" -> "false"'
    ];
    const actual = tokenize(grammar, lines);
    const scopes = ['string.quoted.double.gooo', 'string.quoted.other.backtick.gooo', 'comment.line.double-slash.gooo'];
    for (let index = 0; index < lines.length; index++) {
      const position = lines[index].indexOf('condition_case');
      assertScope(actual[index], position, scopes[index]);
      assert.equal(tokenAt(actual[index], position).scopes.includes('keyword.other.gooo'), false);
    }
  } finally { registry.dispose(); }
});

test('the regression distinguishes the prior grammar with the two words removed', async () => {
  const prior = JSON.parse(source);
  const keywords = prior.repository.keywords.patterns.find(pattern => pattern.name === 'keyword.other.gooo');
  keywords.match = keywords.match.replace('|source_fill|candidate', '');
  const { grammar, registry } = await loadGrammar(JSON.stringify(prior));
  try {
    const line = 'source_fill intent "bounded" { candidate "safe" {} }';
    const actual = tokenize(grammar, [line])[0];
    assert.equal(tokenAt(actual, 0).scopes.includes('keyword.other.gooo'), false);
    assert.equal(tokenAt(actual, line.indexOf('candidate')).scopes.includes('keyword.other.gooo'), false);
  } finally { registry.dispose(); }
});

test('comments and double-quoted strings keep the words out of keyword scopes', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const comment = '// source_fill candidate "candidate"';
    const quoted = String.raw`source_fill intent "candidate \"source_fill\" text" { candidate "safe" {} }`;
    const [commentTokens, quotedTokens] = tokenize(grammar, [comment, quoted]);
    for (const word of ['source_fill', 'candidate']) {
      const position = comment.indexOf(word);
      assertScope(commentTokens, position, 'comment.line.double-slash.gooo');
      assert.equal(tokenAt(commentTokens, position).scopes.includes('keyword.other.gooo'), false);
    }
    assertScope(quotedTokens, 0, 'keyword.other.gooo');
    for (const position of [quoted.indexOf('candidate'), quoted.indexOf('source_fill', 1)]) {
      assertScope(quotedTokens, position, 'string.quoted.double.gooo');
      assert.equal(tokenAt(quotedTokens, position).scopes.includes('keyword.other.gooo'), false);
    }
    assertScope(quotedTokens, quoted.lastIndexOf('candidate'), 'keyword.other.gooo');
  } finally { registry.dispose(); }
});

test('multiline backticks preserve string state and resume assembly keyword scopes', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const lines = [
      'activity Plan(Integer) -> Integer computes `source_fill candidate',
      'candidate source_fill',
      '` assembling { source_fill intent "bounded" { candidate "safe" {} } }'
    ];
    const actual = tokenize(grammar, lines);
    for (let index = 0; index < 2; index++) {
      for (const word of ['source_fill', 'candidate']) {
        const position = lines[index].indexOf(word);
        assertScope(actual[index], position, 'string.quoted.other.backtick.gooo');
        assert.equal(tokenAt(actual[index], position).scopes.includes('keyword.other.gooo'), false);
      }
    }
    for (const word of ['assembling', 'source_fill', 'candidate']) assertScope(actual[2], lines[2].indexOf(word), 'keyword.other.gooo');
  } finally { registry.dispose(); }
});

test('word boundaries exclude underscores and identifier prefix or suffix lookalikes', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const lookalikes = ['source_fill_extra', 'xsource_fill', 'source_filling', 'candidate_count', 'xcandidate', 'candidate2'];
    for (const line of lookalikes) {
      const tokens = tokenize(grammar, [line])[0];
      assert.equal(tokens.some(token => token.scopes.includes('keyword.other.gooo')), false, line);
    }
    for (const line of ['source_fill {', 'candidate "safe"']) assertScope(tokenize(grammar, [line])[0], 0, 'keyword.other.gooo');
  } finally { registry.dispose(); }
});


test('multiline double quotes retain escapes and return to candidate keyword scopes', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const lines = [
      'source_fill intent "candidate',
      String.raw`source_fill candidate \"source_fill\"`,
      '" { candidate "safe" {} }'
    ];
    const actual = tokenize(grammar, lines);
    assertScope(actual[0], 0, 'keyword.other.gooo');
    assertScope(actual[0], lines[0].indexOf('candidate'), 'string.quoted.double.gooo');
    for (const word of ['source_fill', 'candidate']) {
      assertScope(actual[1], lines[1].indexOf(word), 'string.quoted.double.gooo');
      assert.equal(tokenAt(actual[1], lines[1].indexOf(word)).scopes.includes('keyword.other.gooo'), false);
    }
    assertScope(actual[1], lines[1].lastIndexOf('source_fill'), 'string.quoted.double.gooo');
    assert.equal(tokenAt(actual[1], lines[1].lastIndexOf('source_fill')).scopes.includes('keyword.other.gooo'), false);
    assertScope(actual[2], lines[2].indexOf('candidate'), 'keyword.other.gooo');
  } finally { registry.dispose(); }
});

test('existing declaration, assembly, control, and numeric scopes remain available', async () => {
  const { grammar, registry } = await loadGrammar();
  try {
    const lines = [
      'package demo',
      'entity Integer id "urn:integer"',
      'activity Calc(Integer) -> Integer computes',
      'assembling { intent "bounded" hole "boundary" }',
      'if true { return 12 }'
    ];
    const actual = tokenize(grammar, lines);
    assertScope(actual[0], 0, 'keyword.declaration.package.gooo');
    assertScope(actual[1], 0, 'keyword.declaration.package.gooo');
    assertScope(actual[1], lines[1].indexOf('Integer'), 'entity.name.type.gooo');
    assertScope(actual[2], 0, 'keyword.declaration.package.gooo');
    assertScope(actual[2], lines[2].indexOf('Calc'), 'entity.name.function.gooo');
    for (const word of ['assembling', 'intent', 'hole']) assertScope(actual[3], lines[3].indexOf(word), 'keyword.other.gooo');
    assertScope(actual[4], 0, 'keyword.control.gooo');
    assertScope(actual[4], lines[4].indexOf('return'), 'keyword.control.gooo');
    assertScope(actual[4], lines[4].indexOf('true'), 'constant.language.gooo');
    assertScope(actual[4], lines[4].indexOf('12'), 'constant.numeric.gooo');
  } finally { registry.dispose(); }
});
