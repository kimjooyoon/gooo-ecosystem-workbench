# Gooo for Visual Studio Code

## 한국어

`.gooo`, `.gooo.fixture`, `.gooo.template` 파일의 문법 색상과 시작 스니펫,
Gooo 컴파일러 명령을 제공합니다. 설치된 Gooo CLI의 `gooo lsp`를 연결해
진단·자동완성·hover·정의 이동·참조·이름 변경·문서 기호를 제공합니다.

### 설치

1. Visual Studio Code와 Gooo CLI를 설치합니다.
   (`go install github.com/kimjooyoon/meta-ontology-go/cmd/gooo@dev`)
2. 이 폴더에서 `npm ci`와 `npx --yes @vscode/vsce package`를 실행합니다.
3. VS Code의 **Extensions: Install from VSIX...** 메뉴에서 생성된
   `gooo-language-support-0.1.0.vsix`를 선택합니다.

`gooo.compilerPath`로 CLI 경로를 지정할 수 있습니다. Laya를 연결하려면
`gooo.layaUrl`에 로컬 `/v1/systemone` 주소를 설정합니다. 이 값을 비워 두고
`GOOO_LAYA_URL` 환경 변수도 설정하지 않으면 Gooo가 선언된 후보를 결정론적으로
처리합니다. 연결된 Laya도 선언된 후보 안에서만 선택하며, 실행 기록은 새 프로젝트의
`package-execution-receipt.json`에 보관됩니다.

### 명령

- **Gooo: Create and Generate Library**: 2개 패키지 시작 프로젝트를 만들고
  즉시 `gooo package execute`를 호출합니다. 검증된 `generated.gooo`를 편집기로 엽니다.
- **Gooo: Check Current File**: 현재 파일을 저장한 뒤 `gooo check`를 실행합니다.
- **Gooo: Generate Activity Body**: activity 이름을 물어보고 본문 생성을 실행합니다.

스니펫 접두사는 `gooo-package`, `gooo-entity`, `gooo-record`, `gooo-activity`,
`gooo-assembling`, `gooo-bind`입니다. 스니펫은 시작점이며 최종 구문 확인은
`gooo check`가 담당합니다.

## English

This extension makes `.gooo`, `.gooo.fixture`, and `.gooo.template` files
recognizable in VS Code. It starts the installed compiler's `gooo lsp` process
over stdio for inline diagnostics, completion, hover, go to definition,
references, rename, document symbols, and semantic tokens.

## Install from this repository

1. Install Visual Studio Code and the Gooo CLI (`go install github.com/kimjooyoon/meta-ontology-go/cmd/gooo@dev`).
2. From this directory, run `npm ci` and `npx --yes @vscode/vsce package`.
3. In VS Code, run **Extensions: Install from VSIX...** and select `gooo-language-support-0.1.0.vsix`.

Set `gooo.compilerPath` if the executable is not named `gooo` or is not on PATH.
The editor commands save the current buffer before invoking the compiler.

## Commands

- **Gooo: Check Current File** runs `gooo check <file>` and displays compiler output.
- **Gooo: Generate Activity Body** asks for an activity name and runs
  `gooo body-codegen --json --activity <name> <file>`.
- **Gooo: Create and Generate Library** writes the compiler's two-package library
  starter, calls `gooo package execute`, and opens the validated `generated.gooo` source.

The check and body-generation commands save the current editor buffer first.
Their output is shown in the Gooo output panel and does not modify the source
file. Model-assisted assembly remains opt-in through the compiler's existing
model configuration. With no configured provider, generation follows the
compiler's deterministic path.

The library command uses `gooo init --template library` followed immediately by
`gooo package execute`. Set `gooo.layaUrl` to a local Laya `/v1/systemone`
endpoint, or launch VS Code with `GOOO_LAYA_URL` set, and Laya ranks only the
candidate expressions declared in the plan.
Otherwise Gooo picks the first eligible declared candidate deterministically.
The `package-execution-receipt.json` includes the selected Gooo source and
generated native projection. The model does not write arbitrary source or
bypass the package type checks.

## Snippets

Type `gooo-package`, `gooo-entity`, `gooo-record`, `gooo-activity`,
`gooo-assembling`, or `gooo-bind` in a `.gooo` file. Snippets are starting
points; `gooo check` remains the source of truth for accepted syntax.

This is a thin editor client over the compiler's language server. Parsing,
diagnostics, and symbol features stay in Gooo; the extension does not duplicate
language rules or rewrite source while you type.
