const { createHash } = require('node:crypto');
const { spawn } = require('node:child_process');
const fs = require('node:fs/promises');
const os = require('node:os');
const path = require('node:path');
const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let languageClient;

function sourceDigest(source) {
  return `sha256:${createHash('sha256').update(source, 'utf8').digest('hex')}`;
}

function runCompiler(args, title, cwd, options = {}) {
  const quiet = options.quiet === true;
  const executable = vscode.workspace.getConfiguration('gooo').get('compilerPath', 'gooo');
  const environment = { ...process.env };
  const layaUrl = vscode.workspace.getConfiguration('gooo').get('layaUrl', '').trim();
  if (layaUrl) environment.GOOO_LAYA_URL = layaUrl;
  const output = quiet ? undefined : vscode.window.createOutputChannel('Gooo');
  if (output) {
    output.show(true);
    output.appendLine(`$ ${executable} ${args.map((arg) => JSON.stringify(arg)).join(' ')}`);
  }
  let stdoutText = '';
  let stderrText = '';

  return new Promise((resolve) => {
    const child = spawn(executable, args, {
      cwd: cwd ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath,
      env: environment,
      windowsHide: true,
      shell: false
    });
    child.stdout.on('data', (chunk) => {
      const text = chunk.toString();
      stdoutText += text;
      if (output) output.append(text);
    });
    child.stderr.on('data', (chunk) => {
      const text = chunk.toString();
      stderrText += text;
      if (output) output.append(text);
    });
    child.on('error', (error) => {
      if (output) output.appendLine(`\n${error.message}`);
      if (!quiet) void vscode.window.showErrorMessage(`Could not start ${title}: ${error.message}`);
      resolve({ code: 1, stdout: stdoutText, stderr: stderrText });
    });
    child.on('close', (code) => {
      const status = code ?? 1;
      if (output) output.appendLine(`\n${title}: ${status === 0 ? 'finished' : `exit ${status}`}`);
      if (!quiet) {
        if (status === 0) void vscode.window.showInformationMessage(`${title} finished.`);
        else void vscode.window.showErrorMessage(`${title} failed. See the Gooo output.`);
      }
      resolve({ code: status, stdout: stdoutText, stderr: stderrText });
    });
  });
}

async function formatDocument(document) {
  const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'gooo-format-'));
  const tempFile = path.join(tempDir, path.basename(document.uri.fsPath) || 'document.gooo');
  const original = document.getText();
  try {
    await fs.writeFile(tempFile, original, 'utf8');
    const result = await runCompiler(['format', '--json', tempFile], 'Gooo format', tempDir, { quiet: true });
    if (result.code !== 0) {
      throw new Error(result.stderr.trim() || 'the compiler rejected this source for formatting');
    }
    const report = JSON.parse(result.stdout);
    const formatted = typeof report.source === 'string'
      ? report.source
      : report.formatted_digest === sourceDigest('') ? '' : undefined;
    const hasFormattedSource = typeof formatted === 'string';
    const changed = hasFormattedSource && formatted !== original;
    if (report.schema !== 'gooo/format-report/v1' || report.command !== 'format' ||
        report.source_digest !== sourceDigest(original) ||
        !hasFormattedSource || report.formatted_digest !== sourceDigest(formatted) ||
        report.changed !== changed || report.status !== (changed ? 'formatted' : 'canonical') ||
        report.direct_writes !== 0) {
      throw new Error('the compiler returned an incomplete or unexpected format report');
    }
    const end = document.positionAt(original.length);
    return [vscode.TextEdit.replace(new vscode.Range(new vscode.Position(0, 0), end), formatted)];
  } finally {
    await fs.rm(tempDir, { recursive: true, force: true });
  }
}

function currentSource() {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== 'gooo') {
    void vscode.window.showWarningMessage('Open a .gooo source file first.');
    return undefined;
  }
  return editor.document.uri.fsPath;
}

async function createProjectWorkspace(template) {
  const workspace = vscode.workspace.workspaceFolders?.[0];
  if (!workspace) {
    void vscode.window.showWarningMessage('Open a workspace folder before creating a Gooo project.');
    return;
  }
  const folderName = await vscode.window.showInputBox({
    prompt: `Name for the new Gooo ${template} project folder`,
    placeHolder: template === 'library' ? 'my-gooo-library' : 'my-gooo-app',
    validateInput: (value) => value !== '.' && value !== '..' && /^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(value)
      ? undefined
      : 'Use letters, numbers, dot, underscore, or hyphen; start with a letter or number.'
  });
  if (!folderName) return;
  const root = path.join(workspace.uri.fsPath, folderName);
  const created = await runCompiler(['init', '--template', template, root], `Gooo ${template} setup`);
  if (created.code !== 0) return;

  let target = path.join(root, 'main.gooo');
  if (template === 'library') {
    const generated = await runCompiler([
      'package', 'execute', '--json', '--cases', path.join(root, 'cases.json'),
      '--body-plans', path.join(root, 'body-fill-plans.json'),
      path.join(root, 'gooo.workspace.json')
    ], 'Gooo library body generation', root);
    if (generated.stdout.trim()) {
      try {
        await fs.writeFile(path.join(root, 'package-execution-receipt.json'), generated.stdout);
      } catch (error) {
        void vscode.window.showErrorMessage(`Could not save the package execution receipt: ${error.message}`);
        return;
      }
    }
    if (generated.code !== 0) {
      void vscode.window.showWarningMessage(`Library files were created in ${folderName}; generation details are in the Gooo output.`);
      return;
    }
    try {
      const receipt = JSON.parse(generated.stdout);
      const source = receipt.result?.composition?.gooo_source;
      if (typeof source !== 'string' || source.length === 0) {
        throw new Error('the execution receipt did not include generated Gooo source');
      }
      target = path.join(root, 'generated.gooo');
      await fs.writeFile(target, source);
    } catch (error) {
      void vscode.window.showErrorMessage(`Library ran, but its generated Gooo source could not be opened: ${error.message}`);
      return;
    }
  } else {
    const checked = await runCompiler(['check', target], 'Gooo project check', root);
    if (checked.code !== 0) return;
  }
  const document = await vscode.workspace.openTextDocument(target);
  await vscode.window.showTextDocument(document);
}

function activate(context) {
  const compiler = vscode.workspace.getConfiguration('gooo').get('compilerPath', 'gooo');
  const serverOptions = {
    command: compiler,
    args: ['lsp'],
    transport: TransportKind.stdio,
    options: {
      cwd: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath,
      windowsHide: true
    }
  };
  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'gooo' }],
    outputChannelName: 'Gooo Language Server'
  };
  languageClient = new LanguageClient('gooo', 'Gooo Language Server', serverOptions, clientOptions);
  context.subscriptions.push(languageClient);
  void languageClient.start().catch((error) => {
    void vscode.window.showErrorMessage(`Could not start the Gooo language server: ${error.message}`);
  });

  context.subscriptions.push(
    vscode.languages.registerDocumentFormattingEditProvider('gooo', {
      async provideDocumentFormattingEdits(document) {
        try {
          return await formatDocument(document);
        } catch (error) {
          void vscode.window.showErrorMessage(`Gooo format failed: ${error.message}`);
          return [];
        }
      }
    }),
    vscode.commands.registerCommand('gooo.formatCurrentFile', () =>
      vscode.commands.executeCommand('editor.action.formatDocument')),
    vscode.commands.registerCommand('gooo.createProject', async () => {
      const choice = await vscode.window.showQuickPick([
        { label: 'Application', description: 'A runnable Gooo activity project', template: 'app' },
        { label: 'Library', description: 'A package contract with generated activity body', template: 'library' }
      ], { placeHolder: 'Choose a Gooo project type' });
      if (choice) await createProjectWorkspace(choice.template);
    }),
    vscode.commands.registerCommand('gooo.createLibraryWorkspace', () => createProjectWorkspace('library')),
    vscode.commands.registerCommand('gooo.checkCurrentFile', async () => {
      const source = currentSource();
      if (!source) return;
      const document = vscode.window.activeTextEditor.document;
      if (!(await document.save())) return;
      await runCompiler(['check', source], 'Gooo check');
    }),
    vscode.commands.registerCommand('gooo.generateActivityBody', async () => {
      const source = currentSource();
      if (!source) return;
      const document = vscode.window.activeTextEditor.document;
      if (!(await document.save())) return;
      const activity = await vscode.window.showInputBox({
        prompt: 'Activity name declared in this Gooo source file',
        placeHolder: 'Clamp'
      });
      if (!activity) return;
      await runCompiler(['body-codegen', '--json', '--activity', activity, source], 'Gooo body generation');
    })
  );
}

async function deactivate() {
  if (languageClient) await languageClient.stop();
}

module.exports = { activate, deactivate };
