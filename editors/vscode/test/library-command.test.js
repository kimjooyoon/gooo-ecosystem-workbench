'use strict';

const assert = require('node:assert/strict');
const cp = require('node:child_process');
const { createHash } = require('node:crypto');
const fs = require('node:fs');
const fsp = require('node:fs/promises');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');
const { inflateRawSync } = require('node:zlib');

const executable = process.env.GOOO_TEST_LIBRARY_EXECUTABLE || process.env.GOOO_TEST_LSP_EXECUTABLE;
const compilerSHA = process.env.GOOO_TEST_LIBRARY_EXECUTABLE_SHA256 || process.env.GOOO_TEST_LSP_EXECUTABLE_SHA256;
const goExecutable = process.env.GOOO_TEST_LIBRARY_GO_EXECUTABLE;
const goSHA = process.env.GOOO_TEST_LIBRARY_GO_SHA256;
const baselineVSIX = process.env.GOOO_TEST_LIBRARY_BASELINE_VSIX;
const artifactRoot = process.env.GOOO_TEST_LIBRARY_ARTIFACTS;
const required = process.env.GOOO_TEST_LIBRARY_REQUIRED === '1';
const available = Boolean(executable && compilerSHA && goExecutable && goSHA);
const skipNative = available ? false : 'Public compiler/native Go inputs absent; library execution not verified';
const baselineSHA = '5f439f13e1e669c0ebfbf80f25975af6c54455958e2ea8886888e971181a19d6';
const baselineSourceSHA = '2e72290dbbd8ba4c58e33986c5dd7ce6483435d64b92ecd2167c0aaafcdca58a';
const compilerSource = '2b17c4879d0bb2cd1e48ccc23b8743c0fb564267';
const digest = (bytes) => createHash('sha256').update(bytes).digest('hex');
const bound = (bytes) => 'sha256:' + digest(bytes);
const noOp = () => {};
const starterFiles = ['README.md', 'app.gooo', 'cases.json', 'core.gooo', 'gooo.workspace.json'];

function assertInputs() {
  for (const [filename, expected] of [[executable, compilerSHA], [goExecutable, goSHA]]) {
    assert.ok(filename && path.isAbsolute(filename), 'An exact absolute executable is required');
    assert.match(expected || '', /^[a-f0-9]{64}$/);
    const stat = fs.statSync(filename);
    assert.ok(stat.isFile() && stat.size < 64 * 1024 * 1024);
    assert.equal(digest(fs.readFileSync(filename)), expected);
  }
}

// Read only two members from a hash-bound public VSIX. No extraction, archive
// dependency, reconstructed baseline or modified current extension is used.
function zipMember(bytes, member) {
  const end = bytes.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]));
  assert.ok(end >= Math.max(0, bytes.length - 65557));
  assert.equal(end + 22 + bytes.readUInt16LE(end + 20), bytes.length);
  assert.equal(bytes.readUInt32LE(end + 4), 0, 'Single-disk ZIP required');
  const count = bytes.readUInt16LE(end + 10);
  assert.equal(bytes.readUInt16LE(end + 8), count);
  assert.ok(count > 0 && count < 1024);
  const start = bytes.readUInt32LE(end + 16), size = bytes.readUInt32LE(end + 12);
  assert.equal(start + size, end);
  let offset = start, found;
  for (let index = 0; index < count; index++) {
    assert.equal(bytes.readUInt32LE(offset), 0x02014b50);
    const method = bytes.readUInt16LE(offset + 10);
    const compressed = bytes.readUInt32LE(offset + 20), length = bytes.readUInt32LE(offset + 24);
    const nameLength = bytes.readUInt16LE(offset + 28), extra = bytes.readUInt16LE(offset + 30);
    const comment = bytes.readUInt16LE(offset + 32), local = bytes.readUInt32LE(offset + 42);
    const name = bytes.subarray(offset + 46, offset + 46 + nameLength).toString('utf8');
    if (name === member) {
      assert.equal(found, undefined, 'Unique requested member required');
      assert.ok(length > 0 && length <= 128 * 1024);
      assert.equal(bytes.readUInt32LE(local), 0x04034b50);
      assert.equal(bytes.readUInt16LE(local + 8), method);
      const data = local + 30 + bytes.readUInt16LE(local + 26) + bytes.readUInt16LE(local + 28);
      assert.ok(data + compressed <= start);
      const packed = bytes.subarray(data, data + compressed);
      assert.ok(method === 0 || method === 8);
      found = method === 0 ? packed : inflateRawSync(packed, { maxOutputLength: length });
      assert.equal(found.length, length);
    }
    offset += 46 + nameLength + extra + comment;
  }
  assert.equal(offset, end);
  assert.ok(found, 'Public archive member absent: ' + member);
  return found;
}

function baselineSource() {
  assert.ok(baselineVSIX && path.isAbsolute(baselineVSIX));
  const stat = fs.statSync(baselineVSIX);
  assert.ok(stat.isFile() && stat.size < 8 * 1024 * 1024);
  const bytes = fs.readFileSync(baselineVSIX);
  assert.equal(digest(bytes), baselineSHA, 'Use the retained exact public0.2.3 VSIX');
  assert.equal(JSON.parse(zipMember(bytes, 'extension/package.json')).version, '0.2.3');
  const source = zipMember(bytes, 'extension/extension.js');
  assert.equal(digest(source), baselineSourceSHA);
  return source.toString('utf8');
}

function controlledEnvironment() {
  const env = { ...process.env, GOTOOLCHAIN: 'local', GOWORK: 'off',
    PATH: path.dirname(goExecutable) + path.delimiter + (process.env.PATH || '') };
  delete env.GOOO_LAYA_URL;
  delete env.GOOO_LAYA_API_KEY;
  return env;
}

// Compiler output is never supplied by the test. Every record below comes from
// a real child process; the host shim only supplies dialogs and document tabs.
function owner() {
  const active = new Set(), records = [];
  let closed = false;
  function terminate(record) {
    record.child.kill('SIGTERM');
    clearTimeout(record.force);
    record.force = setTimeout(() => {
      if (!record.row.closed) record.child.kill('SIGKILL');
    }, 2000);
    record.force.unref();
  }
  function spawn(command, args, options = {}) {
    assert.equal(closed, false);
    assert.equal(command, executable);
    assert.equal(options.shell, false);
    assert.equal(options.env.GOOO_LAYA_URL, undefined);
    assert.equal(options.env.GOOO_LAYA_API_KEY, undefined);
    const child = cp.spawn(command, args, options);
    const row = { command, args: Array.from(args), cwd: options.cwd, stdout: '', stderr: '', closed: false };
    const record = { child, row };
    active.add(record); records.push(row);
    record.close = new Promise((resolve, reject) => {
      child.once('error', reject);
      child.once('close', (code, signal) => {
        row.closed = true; row.code = code; row.signal = signal;
        clearTimeout(record.force);
        active.delete(record);
        try {
          if (code === 0 && args[0] === 'init') row.initial_sources = starterSnapshot(args[args.length - 1]);
          resolve({ code, signal });
        } catch (error) { row.snapshot_failure = String(error); reject(error); }
      });
    });
    record.close.catch(noOp);
    const timer = setTimeout(() => { row.timeout = true; terminate(record); }, 120000);
    timer.unref();
    record.close.finally(() => clearTimeout(timer)).catch(noOp);
    for (const field of ['stdout', 'stderr']) child[field].on('data', (chunk) => {
      row[field] += chunk.toString('utf8');
      if (Buffer.byteLength(row[field]) > 8 * 1024 * 1024) {
        row.output_limit = true; terminate(record);
      }
    });
    return child;
  }
  async function run(args, cwd) {
    const child = spawn(executable, args, { cwd, env: controlledEnvironment(), shell: false, windowsHide: true });
    const row = records[records.length - 1];
    await Array.from(active).find(record => record.child === child).close;
    assert.equal(row.timeout, undefined);
    assert.equal(row.output_limit, undefined);
    assert.equal(row.signal, null);
    return row;
  }
  async function finish() {
    closed = true;
    await Promise.all(Array.from(active).map(async (record) => {
      terminate(record);
      let timer;
      try {
        await Promise.race([record.close, new Promise((_, reject) => {
          timer = setTimeout(() => reject(new Error('Compiler reap timed out')), 2000);
        })]);
      } catch {
        record.child.kill('SIGKILL');
        await Promise.race([record.close.catch(noOp), new Promise((_, reject) => {
          timer = setTimeout(() => reject(new Error('Compiler forced reap timed out')), 2000);
        })]);
      } finally { clearTimeout(timer); clearTimeout(record.force); }
    }));
    assert.equal(active.size, 0);
  }
  function cancel() { closed = true; for (const record of active) terminate(record); }
  return { spawn, run, finish, cancel, records };
}

function host(source, root, folderName, processes) {
  const callbacks = new Map(), opened = [], messages = [];
  const disposable = () => ({ dispose: noOp });
  const channel = () => ({ dispose: noOp, show: noOp, append: noOp, appendLine: noOp, clear: noOp });
  const vscode = {
    workspace: {
      workspaceFolders: [{ uri: { fsPath: root } }],
      getConfiguration() { return { get: (key, fallback) => key === 'compilerPath' ? executable : fallback }; },
      onDidChangeConfiguration: disposable,
      async openTextDocument(filename) { return { uri: { fsPath: filename }, getText: () => fs.readFileSync(filename, 'utf8') }; }
    },
    window: {
      createOutputChannel: channel,
      async showInputBox(options) { assert.equal(options.validateInput(folderName), undefined); return folderName; },
      async showQuickPick(choices) { return choices.find(choice => choice.template === 'library'); },
      async showTextDocument(document) { opened.push(document.uri.fsPath); },
      showErrorMessage(message) { messages.push({ kind: 'error', message }); },
      showWarningMessage(message) { messages.push({ kind: 'warning', message }); },
      showInformationMessage(message) { messages.push({ kind: 'information', message }); }
    },
    languages: { registerDocumentFormattingEditProvider: disposable },
    commands: { registerCommand(name, callback) { callbacks.set(name, callback); return disposable(); } }
  };
  const imports = {
    vscode, 'vscode-languageclient/node': { LanguageClient: class {} },
    'node:crypto': require('node:crypto'), 'node:path': path, 'node:os': os,
    'node:child_process': { spawn: processes.spawn }, 'node:fs/promises': fsp,
    './body-generation': {},
    './language-server': { createLanguageServerController: () => ({ updateCompiler: noOp, dispose: noOp }) }
  };
  const context = { module: { exports: {} }, process: { env: controlledEnvironment() },
    require(name) { assert.ok(Object.hasOwn(imports, name), 'Unexpected extension import: ' + name); return imports[name]; } };
  vm.createContext(context);
  vm.runInContext(source, context, { filename: 'actual-extension.js', timeout: 1000 });
  context.module.exports.activate({ subscriptions: [] });
  return { callbacks, opened, messages, context };
}

async function ownedProject(t, operation) {
  assertInputs();
  if (artifactRoot) { assert.ok(path.isAbsolute(artifactRoot)); fs.mkdirSync(artifactRoot, { recursive: true }); }
  const root = fs.mkdtempSync(path.join(artifactRoot || os.tmpdir(), 'gooo-library-'));
  const processes = owner();
  const abort = () => processes.cancel();
  t.signal.addEventListener('abort', abort, { once: true });
  try { await operation(root, processes); }
  finally {
    t.signal.removeEventListener('abort', abort);
    await processes.finish();
    fs.writeFileSync(path.join(root, 'command-observations.json'), JSON.stringify(processes.records, null, 2) + '\n');
    t.diagnostic(JSON.stringify({ scope: 'actual-public-compiler-library-with-simulated-editor-host',
      actual_ide_activation: false, compiler_sha256: compilerSHA, go_sha256: goSHA,
      evidence_root: artifactRoot ? root : undefined, commands: processes.records.map(row => ({
        args: row.args, code: row.code, signal: row.signal, stdout_sha256: digest(row.stdout),
        stderr_sha256: digest(row.stderr), timeout: row.timeout, output_limit: row.output_limit
      })) }));
    if (!artifactRoot) fs.rmSync(root, { recursive: true, force: true });
  }
}

function starterSnapshot(root) {
  return Object.fromEntries(starterFiles.map(name => [name, digest(fs.readFileSync(path.join(root, name)))]));
}

function inspectReceipt(receipt, root) {
  assert.equal(receipt.schema, 'gooo/workspace-body-execution-receipt/v1');
  assert.equal(receipt.decision, 'PASS'); assert.equal(receipt.error, undefined);
  assert.equal(receipt.manifest_digest, bound(fs.readFileSync(path.join(root, 'gooo.workspace.json'))));
  assert.equal(receipt.cases_digest, bound(fs.readFileSync(path.join(root, 'cases.json'))));
  const result = receipt.result, composition = result.composition, runtime = result.runtime;
  assert.equal(result.schema, 'gooo/workspace-body-execution/v1');
  assert.equal(result.program.entry.package_path, 'boundedint/app'); assert.equal(result.program.entry.activity, 'Main');
  assert.equal(result.body_fills.length, 1);
  assert.equal(result.body_fills[0].activity.package_path, 'boundedint/core');
  assert.equal(result.body_fills[0].activity.activity, 'Normalize');
  assert.equal(result.body_fills[0].generation.report.body_fill.selected_candidate_id, 'add_one');
  assert.equal(result.body_fills[0].external_plan, undefined);
  assert.equal(result.body_fills[0].input_source_sha256, bound(result.program.lowered_gooo_source));
  assert.equal(result.execution_source_sha256, bound(result.body_fills[0].generation.gooo_source));
  assert.equal(composition.schema, 'gooo/body-composition/v1'); assert.equal(composition.stage, 'COMPLETE');
  assert.ok(composition.gooo_source && !composition.gooo_source.includes('__GOOO_BODY_HOLE_'));
  assert.equal(composition.selected_source_sha256, bound(composition.gooo_source));
  assert.equal(composition.generated_sha256, bound(composition.source));
  assert.equal(composition.driver_sha256, bound(composition.driver_source));
  assert.equal(composition.original_source_sha256, result.execution_source_sha256);
  assert.equal(runtime.schema, 'gooo/body-composition-runtime/v1'); assert.equal(runtime.stage, 'COMPLETE');
  assert.equal(runtime.projection_replayed, true); assert.equal(runtime.runtime_replayed, true);
  assert.equal(runtime.finite_total, 9); assert.equal(runtime.finite_passed, 9);
  assert.equal(runtime.producer_source_sha, compilerSource);
  assert.equal(runtime.go_tool_sha256, 'sha256:' + goSHA);
  assert.match(runtime.go_version, /^go version go1\.27\.2 (darwin|linux)\/(arm64|amd64)$/);
  assert.equal(runtime.generated_sha256, composition.generated_sha256);
  assert.equal(runtime.driver_sha256, composition.driver_sha256);
  assert.equal(runtime.typed_plan_sha256, composition.plan.typed_plan_sha256);
  assert.equal(runtime.selected_source_sha256, composition.selected_source_sha256);
  assert.equal(runtime.model_calls, 0); assert.equal(runtime.runs.length, 2);
  for (const run of runtime.runs) {
    assert.equal(run.started, true); assert.equal(run.completed, true); assert.equal(run.exit_code, 0);
    assert.equal(run.canceled, false); assert.equal(run.timed_out, false); assert.equal(run.output_truncated, false);
  }
  const supplied = JSON.parse(fs.readFileSync(path.join(root, 'cases.json'))).cases;
  const namedOutputs = [
    { 'boundedint/core:Normalize': -3, 'boundedint/core:Clamp': 0, 'boundedint/app:Main': 0 },
    { 'boundedint/core:Normalize': 8, 'boundedint/core:Clamp': 8, 'boundedint/app:Main': 8 },
    { 'boundedint/core:Normalize': 13, 'boundedint/core:Clamp': 10, 'boundedint/app:Main': 10 }
  ];
  assert.deepEqual(supplied.map(row => row.inputs['boundedint/core:Normalize']), [-4, 7, 12]);
  assert.deepEqual(supplied.map(row => row.expected), namedOutputs);
  assert.equal(runtime.traces.length, 3);
  return runtime.traces.map((trace, index) => {
    assert.equal(trace.case_index, index); assert.equal(trace.deliveries.length, 3);
    const outputs = {};
    for (const ref of result.program.activities) {
      const activity = composition.plan.activities.find(value => value.name === ref.lowered_name);
      assert.ok(activity);
      const matches = trace.deliveries.filter(value => value.activity_id === activity.id);
      assert.equal(matches.length, 1);
      const key = ref.package_path + ':' + ref.activity, delivery = matches[0];
      assert.equal(delivery.actual, supplied[index].expected[key]); assert.equal(delivery.passed, true);
      assert.equal(delivery.expected, supplied[index].expected[key]);
      outputs[key] = delivery.actual;
    }
    assert.equal(Object.keys(outputs).length, 3);
    assert.deepEqual(outputs, namedOutputs[index]);
    return outputs;
  });
}

test('library native verification cannot silently skip required inputs', () => {
  if (required) {
    assert.ok(available, 'Required library job needs verified compiler and native Go paths/hashes');
    assert.ok(baselineVSIX, 'Required library job needs retained public baseline');
  }
});

test('retained public0.2.3 library command fails on its absent external plan', {
  skip: available && baselineVSIX ? false : 'Public compiler/Go/baseline absent; old command not executed', timeout: 150000
}, async (t) => ownedProject(t, async (workspace, processes) => {
  const h = host(baselineSource(), workspace, 'old-library', processes);
  await h.callbacks.get('gooo.createLibraryWorkspace')();
  const root = path.join(workspace, 'old-library');
  for (const name of starterFiles) assert.ok(fs.existsSync(path.join(root, name)));
  assert.deepEqual(starterSnapshot(root), processes.records[0].initial_sources);
  assert.equal(fs.existsSync(path.join(root, 'body-fill-plans.json')), false);
  assert.equal(fs.existsSync(path.join(root, 'generated.gooo')), false); assert.deepEqual(h.opened, []);
  assert.equal(processes.records.length, 2); assert.equal(processes.records[0].code, 0);
  assert.equal(processes.records[1].code, 1);
  assert.ok(processes.records[1].args.includes('--body-plans'));
  const raw = fs.readFileSync(path.join(root, 'package-execution-receipt.json'), 'utf8');
  assert.equal(raw, processes.records[1].stdout);
  const receipt = JSON.parse(raw);
  assert.equal(receipt.decision, 'FAIL_CLOSED'); assert.match(receipt.error, /body-fill-plans\.json/);
  assert.equal(receipt.result, undefined);
  t.diagnostic(JSON.stringify({ baseline_vsix_sha256: baselineSHA, baseline_extension_sha256: baselineSourceSHA,
    old_decision: receipt.decision, old_error: receipt.error, generated_source: false }));
}));

for (const route of ['gooo.createProject', 'gooo.createLibraryWorkspace']) {
  test('actual ' + route + ' creates, executes and replays a usable Gooo library', {
    skip: skipNative, timeout: 300000
  }, async (t) => ownedProject(t, async (workspace, processes) => {
    const filename = path.join(__dirname, '..', 'extension.js'), source = fs.readFileSync(filename, 'utf8');
    const h = host(source, workspace, 'usable-library', processes);
    await h.callbacks.get(route)();
    const root = path.join(workspace, 'usable-library');
    assert.equal(processes.records.length, 2);
    for (const row of processes.records) { assert.equal(row.code, 0); assert.equal(row.signal, null); assert.equal(row.timeout, undefined); }
    assert.deepEqual(processes.records[0].args, ['init', '--template', 'library', root]);
    assert.deepEqual(processes.records[1].args, ['package', 'execute', '--json', '--cases',
      path.join(root, 'cases.json'), path.join(root, 'gooo.workspace.json')]);
    assert.equal(fs.existsSync(path.join(root, 'body-fill-plans.json')), false);
    const snapshot = processes.records[0].initial_sources;
    assert.deepEqual(starterSnapshot(root), snapshot);
    const raw = fs.readFileSync(path.join(root, 'package-execution-receipt.json'), 'utf8');
    assert.equal(raw, processes.records[1].stdout);
    const receipt = JSON.parse(raw), outputs = inspectReceipt(receipt, root);
    const generated = path.join(root, 'generated.gooo');
    assert.equal(fs.readFileSync(generated, 'utf8'), receipt.result.composition.gooo_source);
    assert.deepEqual(h.opened, [generated]);
    assert.equal(h.messages.some(value => value.kind === 'error' || value.kind === 'warning'), false);
    assert.ok(h.messages.some(value => value.kind === 'information' &&
      /library.*all supplied cases passed/i.test(value.message)));
    assert.equal(h.messages.some(value => value.message === 'Gooo library body generation finished.'), false);
    const checked = await processes.run(['check', generated], root); assert.equal(checked.code, 0);
    const replayed = await processes.run(['package', 'replay', '--json', '--receipt',
      path.join(root, 'package-execution-receipt.json'), '--cases', path.join(root, 'cases.json'),
      path.join(root, 'gooo.workspace.json')], root);
    assert.equal(replayed.code, 0);
    fs.writeFileSync(path.join(root, 'package-replay-receipt.json'), replayed.stdout);
    const replay = JSON.parse(replayed.stdout);
    assert.equal(replay.replayed_from_sha256, bound(raw));
    assert.deepEqual(inspectReceipt(replay, root), outputs);
    assert.deepEqual(replay.result.program, receipt.result.program);
    assert.deepEqual(replay.result.body_fills, receipt.result.body_fills);
    assert.deepEqual(replay.result.composition, receipt.result.composition);
    assert.equal(replay.result.replay.body_fills_replayed, 1); assert.equal(replay.result.replay.model_calls, 0);
    assert.deepEqual(starterSnapshot(root), snapshot);
    assert.equal(h.context.libraryGeneratedSource(raw), receipt.result.composition.gooo_source);
    const mutations = [
      value => { value.decision = 'PROGRESS'; }, value => { value.schema = 'wrong'; },
      value => { value.result.composition.failure = 'changed'; },
      value => { value.result.runtime.runtime_replayed = false; },
      value => { value.result.runtime.finite_passed -= 1; },
      value => { value.result.runtime.runs[0].exit_code = 1; }
    ];
    for (const mutate of mutations) {
      const altered = JSON.parse(raw); mutate(altered);
      assert.throws(() => h.context.libraryGeneratedSource(JSON.stringify(altered)), /complete passing library result/);
    }
    t.diagnostic(JSON.stringify({ route, extension_sha256: digest(source), named_outputs: outputs,
      receipt_sha256: digest(raw), replay_sha256: digest(replayed.stdout),
      generated_gooo_sha256: digest(receipt.result.composition.gooo_source),
      source_check_exit: checked.code, validator_only_actual_receipt_mutations_refused: mutations.length,
      actual_ide_activation: false }));
  }));
}
