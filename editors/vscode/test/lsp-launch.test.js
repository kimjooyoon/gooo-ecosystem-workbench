'use strict';

const assert = require('node:assert/strict');
const cp = require('node:child_process');
const { createHash } = require('node:crypto');
const { EventEmitter } = require('node:events');
const fs = require('node:fs');
const { createRequire } = require('node:module');
const path = require('node:path');
const { PassThrough } = require('node:stream');
const test = require('node:test');
const vm = require('node:vm');

const executable = process.env.GOOO_TEST_LSP_EXECUTABLE;
const expectedExecutableSHA = process.env.GOOO_TEST_LSP_EXECUTABLE_SHA256;
const required = process.env.GOOO_TEST_LSP_REQUIRED === '1';
const nativeSkip = executable ? false : 'Public compiler absent; native LSP launch was not verified';
const clientSourceSHA = '2a9a3df022e76edff94023c12af6559eb04adf9adc113ab48a798c127226ebff';
const fixture = 'package billing\nnamespace billing\nentity Order id "billing://entity/order"\n' +
  'activity PayOrder(Order) -> Order\n';
const malformed = 'package billing\nnamespace billing\nentity Order id "unterminated';
const uri = 'file:///gooo-lsp-launch.gooo';
const digest = (bytes) => createHash('sha256').update(bytes).digest('hex');
const noOp = () => {};

// Evaluate the whole installed module, not an extracted or copied transport
// function. Only the VS Code host, base client, and unused host API re-exports
// are stubbed. Its protocol streams, semver checks and working-dir code are real.
function loadClient(spawn) {
  const filename = require.resolve('vscode-languageclient/node');
  const source = fs.readFileSync(filename);
  assert.equal(digest(source), clientSourceSHA, 'Review the transport harness when the locked client changes');
  const localRequire = createRequire(filename);
  assert.equal(localRequire('../../package.json').version, '10.1.2');
  class BaseLanguageClient {
    constructor(_id, _name, options) {
      this.clientOptions = options;
      this.outputChannel = { appendLine: noOp, info: noOp, error: noOp };
    }
    info() {}
    error() {}
  }
  const moduleObject = { exports: {} };
  const actualImports = new Set(['fs', 'path', 'readline', '../common/utils/is', './processes',
    'vscode-languageserver-protocol/node', 'semver/functions/parse', 'semver/functions/satisfies']);
  vm.runInNewContext(source.toString('utf8'), {
    module: moduleObject, exports: moduleObject.exports, process, setTimeout, setImmediate,
    require(name) {
      if (name === 'child_process') return { ...cp, spawn };
      if (name === 'vscode') return { version: '1.91.0', workspace: { workspaceFolders: [] } };
      if (name === '../common/client') return { BaseLanguageClient };
      if (name === '../common/api') return {};
      assert.ok(actualImports.has(name), 'Unexpected language-client import: ' + name);
      return localRequire(name);
    }
  }, { filename, timeout: 1000 });
  return { ...moduleObject.exports, protocol: localRequire('vscode-languageserver-protocol/node') };
}

// Capture the real extension's server options with a simulated editor host.
// No extension command, compiler, filesystem operation or real IDE is activated.
function extensionOptions(command) {
  let captured;
  const disposable = () => ({ dispose: noOp });
  const forbidden = () => { throw new Error('Unrelated extension operation'); };
  const vscode = {
    workspace: {
      workspaceFolders: [{ uri: { fsPath: path.join(__dirname, '..') } }],
      getConfiguration() { return { get: (_key, fallback) => command || fallback }; },
      onDidChangeConfiguration: disposable
    },
    window: { createOutputChannel: () => ({ dispose: noOp }), showErrorMessage: forbidden },
    languages: { registerDocumentFormattingEditProvider: disposable },
    commands: { registerCommand: disposable }
  };
  class LanguageClient {
    constructor(_id, _name, options) { captured = options; }
  }
  const stubs = {
    vscode,
    'vscode-languageclient/node': { LanguageClient, TransportKind: { stdio: 0 } },
    'node:crypto': require('node:crypto'), 'node:path': path,
    'node:child_process': { spawn: forbidden },
    'node:fs/promises': {}, 'node:os': {}, './body-generation': {},
    './language-server': {
      createLanguageServerController(createClient) {
        return { updateCompiler: (value) => { createClient(value); }, dispose: noOp };
      }
    }
  };
  const filename = path.join(__dirname, '..', 'extension.js');
  const moduleObject = { exports: {} };
  vm.runInNewContext(fs.readFileSync(filename, 'utf8'), {
    module: moduleObject, exports: moduleObject.exports, process: { env: {} },
    require(name) { assert.ok(Object.hasOwn(stubs, name)); return stubs[name]; }
  }, { filename, timeout: 1000 });
  moduleObject.exports.activate({ subscriptions: [] });
  assert.ok(captured, 'Extension must construct its server configuration');
  return captured;
}

function bounded(promise, label, milliseconds = 5000) {
  let timer;
  return Promise.race([promise, new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(label + ' timed out')), milliseconds);
  })]).finally(() => clearTimeout(timer));
}

function assertNativeInput() {
  assert.ok(path.isAbsolute(executable), 'Use the exact verified compiler executable path');
  assert.match(expectedExecutableSHA || '', /^[a-f0-9]{64}$/, 'A verified compiler SHA256 is required');
  const stats = fs.statSync(executable);
  assert.ok(stats.isFile() && stats.size <= 64 * 1024 * 1024, 'Compiler must be a bounded regular file');
  assert.equal(digest(fs.readFileSync(executable)), expectedExecutableSHA);
}

async function launch(options) {
  const records = [];
  let retired = false;
  const library = loadClient((command, args, spawnOptions) => {
    if (retired) throw new Error('Compiler launch was retired before transport creation completed');
    assert.equal(command, executable);
    assert.equal(spawnOptions.shell, undefined);
    assert.equal(records.length, 0, 'Exactly one owned compiler process per launch');
    const child = cp.spawn(command, args, spawnOptions);
    const record = { child, args: Array.from(args), stderr: '', stdoutBytes: 0, closed: false };
    record.close = new Promise((resolve, reject) => {
      child.once('error', reject);
      child.once('close', (code, signal) => { record.closed = true; resolve({ code, signal }); });
    });
    // Attach rejection handling immediately, including a spawn failure.
    record.close.catch(noOp);
    child.stderr.on('data', (chunk) => {
      record.stderr += chunk.toString('utf8');
      if (Buffer.byteLength(record.stderr) > 1024 * 1024) child.kill('SIGKILL');
    });
    child.stdout.on('data', (chunk) => {
      record.stdoutBytes += chunk.length;
      if (record.stdoutBytes > 1024 * 1024) child.kill('SIGKILL');
    });
    records.push(record);
    return child;
  });
  const client = new library.LanguageClient('gooo-test', 'Gooo transport test', options, {
    stdioOptions: { stdout: noOp, stderr: noOp }
  });
  let transports;
  try {
    transports = await bounded(client.createMessageTransports('utf-8'), 'Create transports');
    assert.equal(records.length, 1);
    return { ...records[0], record: records[0], transports, protocol: library.protocol };
  } catch (error) {
    retired = true;
    if (records[0]) await cleanup({ record: records[0], transports });
    throw error;
  }
}

async function cleanup(session) {
  session.transports?.reader.dispose();
  session.transports?.writer.dispose();
  const record = session.record;
  if (!record.closed) {
    record.child.stdin.destroy();
    record.child.kill('SIGTERM');
    try { await bounded(record.close, 'Owned process termination', 2000); }
    catch { record.child.kill('SIGKILL'); await bounded(record.close, 'Owned process reap', 2000); }
  }
}

test('real locked client preserves the extension executable arguments without a transport flag', async () => {
  const options = extensionOptions('/verified/gooo');
  assert.equal(options.transport, undefined);
  assert.deepEqual(Array.from(options.args), ['lsp']);
  const spawns = [];
  const library = loadClient((command, args) => {
    spawns.push({ command, args: Array.from(args) });
    const child = new EventEmitter();
    child.pid = 1;
    child.stdin = new PassThrough(); child.stdout = new PassThrough(); child.stderr = new PassThrough();
    return child;
  });
  const client = new library.LanguageClient('gooo-test', 'Gooo transport test', options, {
    stdioOptions: { stdout: noOp, stderr: noOp }
  });
  const transports = await client.createMessageTransports('utf-8');
  transports.reader.dispose(); transports.writer.dispose();
  assert.deepEqual(spawns, [{ command: '/verified/gooo', args: ['lsp'] }]);
});

test('native LSP compiler inputs are required by the owned verification job', () => {
  if (required) assert.ok(executable && expectedExecutableSHA, 'Required native verification cannot silently skip');
});

test('old explicit stdio transport adds a flag rejected by the public compiler', {
  skip: nativeSkip, timeout: 15000
}, async (t) => {
  assertNativeInput();
  const options = extensionOptions(executable);
  const session = await launch({ ...options, transport: 0 });
  try {
    const result = await bounded(session.record.close, 'Old transport refusal');
    assert.deepEqual(session.record.args, ['lsp', '--stdio']);
    assert.deepEqual(result, { code: 2, signal: null });
    assert.equal(session.record.stderr.trim(), 'usage: gooo lsp');
    assert.equal(session.record.stdoutBytes, 0);
    t.diagnostic(JSON.stringify({ scope: 'actual-owned-compiler-old-transport',
      compiler_sha256: expectedExecutableSHA, args: session.record.args,
      process: result, stderr: session.record.stderr, stdout_bytes: session.record.stdoutBytes }));
  } finally { await cleanup(session); }
});

test('extension launch reaches public compiler diagnostics and completion, then shuts down', {
  skip: nativeSkip, timeout: 25000
}, async (t) => {
  assertNativeInput();
  const options = extensionOptions(executable);
  assert.equal(options.transport, undefined);
  const session = await launch(options);
  const connection = session.protocol.createMessageConnection(session.transports.reader,
    session.transports.writer, { error: noOp, warn: noOp, info: noOp, log: noOp });
  const notifications = [], waiters = [];
  connection.onNotification('textDocument/publishDiagnostics', (params) => {
    const waiter = waiters.shift();
    if (waiter) waiter(params); else notifications.push(params);
  });
  const nextDiagnostic = () => bounded(notifications.length ? Promise.resolve(notifications.shift()) :
    new Promise((resolve) => waiters.push(resolve)), 'Publish diagnostics');
  connection.listen();
  try {
    assert.deepEqual(session.record.args, ['lsp']);
    const initialized = await bounded(connection.sendRequest('initialize', { rootUri: null }), 'Initialize');
    assert.equal(initialized.serverInfo.name, 'gooo-lsp');
    assert.equal(initialized.serverInfo.version, 'current-ddaf');
    assert.deepEqual(initialized.capabilities.textDocumentSync, { openClose: true, change: 2 });
    assert.ok(initialized.capabilities.completionProvider);
    await connection.sendNotification('initialized', {});
    await connection.sendNotification('textDocument/didOpen', {
      textDocument: { uri, languageId: 'gooo', version: 1, text: fixture }
    });
    const opened = await nextDiagnostic();
    assert.equal(opened.uri, uri); assert.deepEqual(opened.diagnostics, []);
    const completion = await bounded(connection.sendRequest('textDocument/completion', {
      textDocument: { uri }, position: { line: 3, character: 14 }
    }), 'Completion');
    const labels = completion.items.map((item) => item.label);
    assert.ok(labels.includes('Order')); assert.ok(labels.includes('id'));
    await connection.sendNotification('textDocument/didChange', {
      textDocument: { uri, version: 2 }, contentChanges: [{ text: malformed }]
    });
    const changed = await nextDiagnostic();
    assert.equal(changed.uri, uri);
    assert.ok(changed.diagnostics.some((diagnostic) => diagnostic.code === 'lex.unterminated-string'));
    const shutdown = await bounded(connection.sendRequest('shutdown'), 'Shutdown');
    assert.equal(shutdown, null);
    await connection.sendNotification('exit');
    const result = await bounded(session.record.close, 'Normal process exit');
    assert.deepEqual(result, { code: 0, signal: null });
    assert.equal(session.record.stderr, '');
    assert.ok(session.record.stdoutBytes > 0 && session.record.stdoutBytes <= 1024 * 1024);
    t.diagnostic(JSON.stringify({ scope: 'actual-owned-compiler-with-simulated-editor-host',
      compiler_sha256: expectedExecutableSHA, args: session.record.args, initialize: initialized,
      opened, completion, completion_labels_verified: ['Order', 'id'], changed,
      shutdown_result: shutdown, process: result, stderr: session.record.stderr,
      stdout_bytes: session.record.stdoutBytes, actual_ide_activation: false }));
  } finally { connection.dispose(); await cleanup(session); }
});
