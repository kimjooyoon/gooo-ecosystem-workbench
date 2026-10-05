const { spawn } = require('node:child_process');
const fs = require('node:fs/promises');
const path = require('node:path');
const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let languageClient;

function runCompiler(args, title, cwd) {
  const executable = vscode.workspace.getConfiguration('gooo').get('compilerPath', 'gooo');
  const environment = { ...process.env };
  const layaUrl = vscode.workspace.getConfiguration('gooo').get('layaUrl', '').trim();
  if (layaUrl) environment.GOOO_LAYA_URL = layaUrl;
  const output = vscode.window.createOutputChannel('Gooo');
  output.show(true);
  output.appendLine(`$ ${executable} ${args.map((arg) => JSON.stringify(arg)).join(' ')}`);
  let stdoutText = '';

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
      output.append(text);
    });
    child.stderr.on('data', (chunk) => output.append(chunk.toString()));
    child.on('error', (error) => {
      output.appendLine(`\n${error.message}`);
      void vscode.window.showErrorMessage(`Could not start ${title}: ${error.message}`);
      resolve({ code: 1, stdout: stdoutText });
    });
    child.on('close', (code) => {
      const status = code ?? 1;
      output.appendLine(`\n${title}: ${status === 0 ? 'finished' : `exit ${status}`}`);
      if (status === 0) void vscode.window.showInformationMessage(`${title} finished.`);
      else void vscode.window.showErrorMessage(`${title} failed. See the Gooo output.`);
      resolve({ code: status, stdout: stdoutText });
    });
  });
}

function currentSource() {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== 'gooo') {
    void vscode.window.showWarningMessage('Open a .gooo source file first.');
    return undefined;
  }
  return editor.document.uri.fsPath;
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
    vscode.commands.registerCommand('gooo.createLibraryWorkspace', async () => {
      const workspace = vscode.workspace.workspaceFolders?.[0];
      if (!workspace) {
        void vscode.window.showWarningMessage('Open a workspace folder before creating a Gooo library.');
        return;
      }
      const folderName = await vscode.window.showInputBox({
        prompt: 'Name for the new Gooo library folder',
        placeHolder: 'my-gooo-library',
        validateInput: (value) => value !== '.' && value !== '..' && /^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(value)
          ? undefined
          : 'Use letters, numbers, dot, underscore, or hyphen; start with a letter or number.'
      });
      if (!folderName) return;
      const root = path.join(workspace.uri.fsPath, folderName);
      const created = await runCompiler(['init', '--template', 'library', root], 'Gooo library setup');
      if (created.code !== 0) return;
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
      if (generated.code === 0) {
        try {
          const receipt = JSON.parse(generated.stdout);
          const source = receipt.result?.composition?.gooo_source;
          if (typeof source !== 'string' || source.length === 0) {
            throw new Error('the execution receipt did not include generated Gooo source');
          }
          const generatedPath = path.join(root, 'generated.gooo');
          await fs.writeFile(generatedPath, source);
          const document = await vscode.workspace.openTextDocument(generatedPath);
          await vscode.window.showTextDocument(document);
        } catch (error) {
          void vscode.window.showErrorMessage(`Library ran, but its generated Gooo source could not be opened: ${error.message}`);
        }
      } else {
        void vscode.window.showWarningMessage(`Library files were created in ${folderName}; generation details are in the Gooo output.`);
      }
    }),
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
