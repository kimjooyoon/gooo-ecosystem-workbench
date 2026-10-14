'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');
const { createLanguageServerController } = require('../language-server');

function deferred() {
  let resolve, reject;
  const promise = new Promise((ok, fail) => { resolve = ok; reject = fail; });
  return { promise, resolve, reject };
}

// Only fake promises and clients are used. No compiler or VS Code host runs.
function fakeClient(command, events, options = {}) {
  let running = false;
  return {
    async start() {
      events.push('start:' + command);
      if (options.start) await options.start();
      running = true;
      events.push('started:' + command);
    },
    needsStop() { return options.needsStop ? options.needsStop() : running; },
    async dispose() {
      events.push('dispose:' + command);
      if (options.dispose) await options.dispose();
      running = false;
      events.push('disposed:' + command);
    }
  };
}

function controllerHarness(optionsFor = () => ({})) {
  const events = [], errors = [], clients = [];
  const controller = createLanguageServerController((command) => {
    events.push('create:' + command);
    const options = optionsFor(command);
    const client = fakeClient(command, events, options);
    clients.push({ command, client });
    return client;
  }, (message) => errors.push(message));
  return { controller, events, errors, clients };
}

test('failed compiler start is disposed and a corrected path starts without reactivation', async () => {
  const h = controllerHarness((command) => command === '/missing/gooo' ? {
    start: () => Promise.reject(new Error('ENOENT')),
    // LanguageClient 10 StartFailed cleanup rejects without a running connection.
    needsStop: () => false,
    dispose: () => Promise.reject(new Error('Client is not running: startFailed'))
  } : {});
  await h.controller.updateCompiler('/missing/gooo');
  assert.match(h.errors[0], /Could not start.*ENOENT/);
  await h.controller.updateCompiler('/installed/gooo');
  assert.deepEqual(h.clients.map((c) => c.command), ['/missing/gooo', '/installed/gooo']);
  assert.deepEqual(h.events, [
    'create:/missing/gooo', 'start:/missing/gooo', 'dispose:/missing/gooo',
    'create:/installed/gooo', 'start:/installed/gooo', 'started:/installed/gooo'
  ]);
  assert.equal(h.errors.length, 1);
  await h.controller.dispose();
});

test('pending start coalesces rapid changes to only the latest compiler', async () => {
  const start = deferred();
  const h = controllerHarness((command) => command === 'first' ? { start: () => start.promise } : {});
  const first = h.controller.updateCompiler('first');
  await Promise.resolve();
  assert.deepEqual(h.events, ['create:first', 'start:first']);
  const skipped = h.controller.updateCompiler('intermediate');
  const latest = h.controller.updateCompiler('latest');
  assert.equal(h.clients.length, 1);
  start.resolve();
  await Promise.all([first, skipped, latest]);
  assert.deepEqual(h.events, [
    'create:first', 'start:first', 'started:first', 'dispose:first', 'disposed:first',
    'create:latest', 'start:latest', 'started:latest'
  ]);
  assert.deepEqual(h.errors, []);
  await h.controller.dispose();
});

test('queued obsolete updates cannot repeatedly retry the latest failed start', async () => {
  const start = deferred();
  let latestAttempts = 0;
  const h = controllerHarness((command) => {
    if (command === 'first') return { start: () => start.promise };
    if (command === 'latest' && ++latestAttempts === 1) return {
      start: () => Promise.reject(new Error('latest path is missing')),
      needsStop: () => false,
      dispose: () => Promise.reject(new Error('startFailed'))
    };
    return {};
  });
  const opening = h.controller.updateCompiler('first');
  await Promise.resolve();
  const queued = ['second', 'third', 'latest'].map((command) => h.controller.updateCompiler(command));
  start.resolve();
  await Promise.all([opening, ...queued]);
  assert.deepEqual(h.clients.map((c) => c.command), ['first', 'latest']);
  assert.equal(latestAttempts, 1);
  assert.equal(h.errors.length, 1);
  assert.match(h.errors[0], /latest path is missing/);
  await h.controller.updateCompiler('latest');
  assert.deepEqual(h.clients.map((c) => c.command), ['first', 'latest', 'latest']);
  assert.equal(latestAttempts, 2);
  assert.equal(h.errors.length, 1);
  await h.controller.dispose();
});

test('pending disposal finishes before creating the latest requested client', async () => {
  const disposal = deferred();
  const entered = deferred();
  const h = controllerHarness((command) => command === 'first' ? {
    dispose: () => { entered.resolve(); return disposal.promise; }
  } : {});
  await h.controller.updateCompiler('first');
  const obsolete = h.controller.updateCompiler('obsolete');
  await entered.promise;
  const latest = h.controller.updateCompiler('latest');
  assert.deepEqual(h.clients.map((c) => c.command), ['first']);
  disposal.resolve();
  await Promise.all([obsolete, latest]);
  assert.deepEqual(h.events, [
    'create:first', 'start:first', 'started:first', 'dispose:first', 'disposed:first',
    'create:latest', 'start:latest', 'started:latest'
  ]);
  await h.controller.dispose();
});

test('a healthy same-path update leaves its client running', async () => {
  const h = controllerHarness();
  await h.controller.updateCompiler('same');
  await h.controller.updateCompiler('same');
  await h.controller.updateCompiler('same');
  assert.deepEqual(h.events, ['create:same', 'start:same', 'started:same']);
  await h.controller.dispose();
  await h.controller.dispose();
  assert.deepEqual(h.events.slice(3), ['dispose:same', 'disposed:same']);
});

test('closing during start awaits cleanup and ignores every later update', async () => {
  const start = deferred(), disposal = deferred(), entered = deferred();
  const h = controllerHarness(() => ({
    start: () => start.promise,
    dispose: () => { entered.resolve(); return disposal.promise; }
  }));
  const opening = h.controller.updateCompiler('first');
  await Promise.resolve();
  let closed = false;
  const closing = h.controller.dispose().then(() => { closed = true; });
  const ignored = h.controller.updateCompiler('never');
  start.resolve();
  await entered.promise;
  assert.equal(closed, false);
  assert.deepEqual(h.clients.map((c) => c.command), ['first']);
  disposal.resolve();
  await Promise.all([opening, closing, ignored]);
  await h.controller.updateCompiler('also-never');
  assert.equal(closed, true);
  assert.deepEqual(h.events, [
    'create:first', 'start:first', 'started:first', 'dispose:first', 'disposed:first'
  ]);
});

test('a running disposal failure blocks replacement until cleanup succeeds', async () => {
  let refuse = true;
  const h = controllerHarness((command) => command === 'first' ? {
    dispose: () => refuse ? Promise.reject(new Error('cannot stop')) : Promise.resolve()
  } : {});
  await h.controller.updateCompiler('first');
  await h.controller.updateCompiler('blocked');
  assert.deepEqual(h.clients.map((c) => c.command), ['first']);
  assert.match(h.errors[0], /Could not stop.*cannot stop/);
  refuse = false;
  await h.controller.updateCompiler('recovered');
  assert.deepEqual(h.clients.map((c) => c.command), ['first', 'recovered']);
  assert.equal(h.events.filter((event) => event === 'dispose:first').length, 2);
  await h.controller.dispose();
});

test('a failed start that still needs stopping cannot bypass disposal failure', async () => {
  let refuse = true;
  const h = controllerHarness((command) => command === 'failed' ? {
    start: () => Promise.reject(new Error('start refused')),
    needsStop: () => true,
    dispose: () => refuse ? Promise.reject(new Error('connection still active')) : Promise.resolve()
  } : {});
  await h.controller.updateCompiler('failed');
  await h.controller.updateCompiler('blocked');
  assert.deepEqual(h.clients.map((c) => c.command), ['failed']);
  assert.match(h.errors[1], /Could not stop.*connection still active/);
  refuse = false;
  await h.controller.dispose();
});

test('client construction failure reports an error and later updates recover', async () => {
  const h = controllerHarness((command) => {
    if (command === 'bad-construction') throw new Error('invalid command');
    return {};
  });
  await h.controller.updateCompiler('bad-construction');
  assert.deepEqual(h.clients, []);
  assert.match(h.errors[0], /invalid command/);
  await h.controller.updateCompiler('valid');
  assert.deepEqual(h.clients.map((c) => c.command), ['valid']);
  await h.controller.dispose();
});

test('queued obsolete updates do not repeat the latest construction failure', async () => {
  const start = deferred();
  let constructions = 0;
  const h = controllerHarness((command) => {
    if (command === 'first') return { start: () => start.promise };
    if (command === 'bad-construction' && ++constructions === 1) throw new Error('cannot construct');
    return {};
  });
  const opening = h.controller.updateCompiler('first');
  await Promise.resolve();
  const queued = ['obsolete', 'bad-construction'].map((command) => h.controller.updateCompiler(command));
  start.resolve();
  await Promise.all([opening, ...queued]);
  assert.equal(constructions, 1);
  assert.deepEqual(h.clients.map((c) => c.command), ['first']);
  assert.equal(h.errors.length, 1);
  await h.controller.updateCompiler('bad-construction');
  assert.equal(constructions, 2);
  assert.deepEqual(h.clients.map((c) => c.command), ['first', 'bad-construction']);
  await h.controller.dispose();
});

function extensionHarness(optionsFor = () => ({}), initialPath = 'gooo') {
  const events = [], errors = [], clients = [], updates = [], outputChannels = [];
  const outputChannelOptions = [];
  const commands = new Map();
  const settings = { compilerPath: initialPath };
  let configListener, latest = Promise.resolve();
  const output = {
    disposeCalls: 0,
    dispose() { this.disposeCalls++; },
    append() {}, appendLine() {}, clear() {}, show() {}
  };
  const disposable = () => ({ dispose() {} });
  const vscode = {
    workspace: {
      workspaceFolders: [{ uri: { fsPath: '/fake/workspace' } }],
      getConfiguration(section) {
        assert.equal(section, 'gooo');
        return { get(name, fallback) { return settings[name] ?? fallback; } };
      },
      onDidChangeConfiguration(listener) {
        assert.equal(configListener, undefined);
        configListener = listener;
        return disposable();
      }
    },
    window: {
      createOutputChannel(name, options) {
        outputChannels.push(name); outputChannelOptions.push(options); return output;
      },
      showErrorMessage(message) { errors.push(message); return Promise.resolve(); }
    },
    languages: {
      registerDocumentFormattingEditProvider(language, provider) {
        assert.equal(language, 'gooo');
        assert.equal(typeof provider.provideDocumentFormattingEdits, 'function');
        return disposable();
      }
    },
    commands: {
      registerCommand(name, handler) { commands.set(name, handler); return disposable(); },
      executeCommand() { throw new Error('Unrelated command must not execute'); }
    }
  };
  class LanguageClient {
    constructor(id, label, serverOptions, clientOptions) {
      this.serverOptions = serverOptions;
      this.clientOptions = clientOptions;
      this.fake = fakeClient(serverOptions.command, events, optionsFor(serverOptions.command));
      clients.push(this);
      events.push('create:' + serverOptions.command);
      assert.equal(id, 'gooo');
      assert.equal(label, 'Gooo Language Server');
    }
    start() { return this.fake.start(); }
    needsStop() { return this.fake.needsStop(); }
    dispose() { return this.fake.dispose(); }
  }
  const moduleObject = { exports: {} };
  const forbidden = () => { throw new Error('Real compiler or filesystem operation is forbidden'); };
  const stubs = {
    vscode,
    'vscode-languageclient/node': { LanguageClient, TransportKind: { stdio: 'fake-stdio' } },
    'node:crypto': require('node:crypto'),
    'node:child_process': { spawn: forbidden },
    'node:fs/promises': new Proxy({}, { get: () => forbidden }),
    'node:os': { tmpdir: forbidden },
    'node:path': path,
    './body-generation': {
      bodyGenerationSummary: forbidden, canApplyGeneratedSource: forbidden, parseBodyCodegenResult: forbidden
    },
    './language-server': {
      createLanguageServerController(createClient, reportError) {
        const real = createLanguageServerController(createClient, reportError);
        return {
          updateCompiler(command) {
            updates.push(command);
            latest = real.updateCompiler(command);
            return latest;
          },
          dispose() { latest = real.dispose(); return latest; }
        };
      }
    }
  };
  const filename = path.join(__dirname, '..', 'extension.js');
  vm.runInNewContext(fs.readFileSync(filename, 'utf8'), {
    module: moduleObject, exports: moduleObject.exports,
    require(name) {
      assert.ok(Object.hasOwn(stubs, name), 'Unexpected extension import: ' + name);
      return stubs[name];
    },
    process: { env: {} }
  }, { filename });
  return {
    extension: moduleObject.exports, events, errors, clients, updates, outputChannels, outputChannelOptions, output, commands,
    activate() { const context = { subscriptions: [] }; moduleObject.exports.activate(context); return context; },
    settled() { return latest; },
    change(command, affected) {
      settings.compilerPath = command;
      configListener({
        affectsConfiguration(name) { assert.equal(name, 'gooo.compilerPath'); return affected; }
      });
    }
  };
}

test('actual extension activation binds compilerPath changes and ignores unrelated settings', async () => {
  const h = extensionHarness();
  const context = h.activate();
  await h.settled();
  assert.deepEqual(h.updates, ['gooo']);
  const first = h.clients[0];
  assert.equal(first.serverOptions.command, 'gooo');
  assert.deepEqual(Array.from(first.serverOptions.args), ['lsp']);
  assert.equal(Object.hasOwn(first.serverOptions, 'transport'), false);
  assert.equal(first.serverOptions.options.cwd, '/fake/workspace');
  assert.equal(first.serverOptions.options.windowsHide, true);
  assert.deepEqual(JSON.parse(JSON.stringify(first.clientOptions.documentSelector)),
    [{ scheme: 'file', language: 'gooo' }]);
  assert.deepEqual(h.outputChannels, ['Gooo Language Server']);
  assert.deepEqual(JSON.parse(JSON.stringify(h.outputChannelOptions)), [{ log: true }]);
  assert.equal(first.clientOptions.outputChannel, h.output);
  assert.ok(context.subscriptions.includes(h.output));
  assert.ok(h.commands.has('gooo.generateActivityBody'));
  h.change('not-an-update', false);
  await h.settled();
  assert.deepEqual(h.updates, ['gooo']);
  h.change('/installed/gooo', true);
  await h.settled();
  h.change('/installed/gooo', true);
  await h.settled();
  assert.deepEqual(h.clients.map((c) => c.serverOptions.command), ['gooo', '/installed/gooo']);
  await h.extension.deactivate();
  h.change('never-after-deactivation', true);
  await h.settled();
  assert.equal(h.clients.length, 2);
  assert.equal(h.events.filter((event) => event === 'dispose:/installed/gooo').length, 1);
  assert.deepEqual(h.errors, []);
});

test('actual extension corrects a failed configured path and awaits pending cleanup on deactivation', async () => {
  const start = deferred(), startEntered = deferred(), disposal = deferred(), entered = deferred();
  const h = extensionHarness((command) => command === 'missing' ? {
    start: () => Promise.reject(new Error('ENOENT')),
    needsStop: () => false,
    dispose: () => Promise.reject(new Error('startFailed'))
  } : {
    start: () => { startEntered.resolve(); return start.promise; },
    dispose: () => { entered.resolve(); return disposal.promise; }
  }, 'missing');
  h.activate();
  await h.settled();
  assert.match(h.errors[0], /Could not start.*ENOENT/);
  h.change('corrected', true);
  await startEntered.promise;
  assert.deepEqual(h.clients.map((c) => c.serverOptions.command), ['missing', 'corrected']);
  let deactivated = false;
  const closing = h.extension.deactivate().then(() => { deactivated = true; });
  h.change('never', true);
  start.resolve();
  await entered.promise;
  assert.equal(deactivated, false);
  disposal.resolve();
  await closing;
  await h.settled();
  assert.equal(deactivated, true);
  assert.deepEqual(h.clients.map((c) => c.serverOptions.command), ['missing', 'corrected']);
  assert.equal(h.errors.length, 1);
});
