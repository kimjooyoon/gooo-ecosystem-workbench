# 처음 실행하기: API 호출을 재시도할지 판단하기

[English](first-run.md)

기존 Gooo 정책을 실행하고 생성된 Go를 읽은 뒤, 저장한 선택을 다시 실행합니다.
두 번 시도한 일시적 실패에서 직전 대기가 7ms, 상한이 100ms이면
`retry: true`, `delay_ms: 14`, `reason: "retry"`를 돌려줍니다.
매 동작마다 사람이 승인하지 않아도 이 판단을 계산합니다.

## 1. 예제 받기와 Go 선택

Linux x86-64, macOS Apple Silicon 또는 macOS Intel에서 Git, curl, tar와
`sha256sum` 또는 `shasum`이 필요합니다. 네이티브 실행에는 **Go 1.27.2**가
필요합니다. 직접 설치하거나 도구 체인 선택을 지원하는 Go 1.21 이상을 사용합니다.
아래 첫 명령은 1.27.2가 캐시에 없으면 해당 도구 체인을 내려받을 수 있습니다.
모든 명령 묶음을 같은 터미널에서 실행하고, 명령이 실패하면 멈춥니다.

```sh
set -eu
export GOTOOLCHAIN=go1.27.2
go version
git clone --no-checkout --depth 1 https://github.com/kimjooyoon/gooo-ecosystem-workbench.git gooo-workbench
cd gooo-workbench
git fetch --depth 1 origin b94b3adb4845782743a1c1a45071a600824f81e6
git checkout --detach b94b3adb4845782743a1c1a45071a600824f81e6
```

`go version`이 `go1.27.2`를 표시하는지 확인합니다. 예제는 작업장 커밋
`b94b3adb4845782743a1c1a45071a600824f81e6`으로 고정합니다.
[source.gooo](../examples/retry-policy/source.gooo)는 재시도 조건, 상한이 있는
대기 시간 계산, 결과 필드 세 개의 선택을 선언합니다. 다섯 조립 사례로 여덟 조합을
검사합니다. [cases.json](../examples/retry-policy/cases.json)에는 별도 입력 열두 개와
기대 판단이 있습니다.

## 2. 고정한 공개 컴파일러 설치

[Gooo 0.6.23-dev 실험 프리릴리스](https://github.com/kimjooyoon/meta-ontology-go/releases/tag/v0.6.23-dev),
소스 `2b17c4879d0bb2cd1e48ccc23b8743c0fb564267`을 사용합니다.
운영체제에 맞는 실행 파일을 내려받고 고정한 파일 지문을 확인한 뒤, Git이 제외하는
`.compiler/bin`에 설치합니다. 언어 엔진을 직접 빌드하지 않습니다.
재현을 위해 고른 버전이며 안정판이나 최신판이라는 의미는 아닙니다.

```sh
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64)
    GOOO_ASSET=gooo-linux-amd64.tar.gz
    GOOO_ARCHIVE_SHA256=603619c0938cb2b73db2b1420cf74dc1a8d856c311267b781e9687f23e57bae9 ;;
  Darwin/arm64)
    GOOO_ASSET=gooo-darwin-arm64.tar.gz
    GOOO_ARCHIVE_SHA256=0658f1c5f1c710a52f6f6d4d0437b78ac24df77f9d2cab35cea3db20d62fdb04 ;;
  Darwin/x86_64)
    GOOO_ASSET=gooo-darwin-amd64.tar.gz
    GOOO_ARCHIVE_SHA256=54345ca51ca1bc36e96f6047366473d8190b0a739bf4089c1a0a0a0a9718c630 ;;
  *) echo "This guide supports Linux x86-64 and macOS arm64/x86-64." >&2; exit 1 ;;
esac
gooo_check_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum -c -
  else shasum -a 256 -c -; fi
}
GOOO_RELEASE_URL=https://github.com/kimjooyoon/meta-ontology-go/releases/download/v0.6.23-dev
mkdir -p .compiler/downloads .compiler/bin
curl --fail --location --show-error "$GOOO_RELEASE_URL/$GOOO_ASSET" --output ".compiler/downloads/$GOOO_ASSET"
curl --fail --location --show-error "$GOOO_RELEASE_URL/SHA256SUMS" --output .compiler/downloads/SHA256SUMS
(
  cd .compiler/downloads
  printf '%s  SHA256SUMS\n' acfdc5d95098398b69f5278e299fdb4b393f98f4a4d155461c5ab8665347d8cd | gooo_check_sha256
  printf '%s  %s\n' "$GOOO_ARCHIVE_SHA256" "$GOOO_ASSET" | gooo_check_sha256
  awk -v file="$GOOO_ASSET" '$2 == file { print }' SHA256SUMS | gooo_check_sha256
)
tar -xzf ".compiler/downloads/$GOOO_ASSET" -C .compiler/bin
export PATH="$PWD/.compiler/bin:$PATH"
command -v gooo
gooo version --build --json
```

전체 체크섬 파일은 내려받지 않은 다른 파일도 나열하므로, 해당 운영체제 파일의
행만 선택해 검사합니다. 체크섬이 다르면 진행하지 않습니다. 컴파일러 정보에는
`version: "0.6.23-dev"`, `go_version: "go1.27.2"`,
`vcs_revision: "2b17c4879d0bb2cd1e48ccc23b8743c0fb564267"`,
`vcs_modified: "false"`, `decision_runtime.version: "v0.2.26-experimental"`이
있어야 합니다. `command -v gooo`가 표시한 절대 경로가 이 안내의 컴파일러입니다.

## 3. 검사, Go 생성, 판단 읽기

같은 저장소 루트에서 실행합니다.

```sh
GOOO_RUN=out/retry-first-use
mkdir -p out
mkdir "$GOOO_RUN"
gooo version --build --json > "$GOOO_RUN/compiler-build.json"
gooo check --json examples/retry-policy/source.gooo > "$GOOO_RUN/check.json"
gooo body-compose --source examples/retry-policy/source.gooo \
  --cases examples/retry-policy/cases.json \
  --out "$GOOO_RUN/composition" > "$GOOO_RUN/execution.json"
```

모델 옵션을 지정하지 않습니다. Gooo가 선언된 후보를 고정 순서로 검사하고 일반 Go를
생성한 뒤, Go 1.27.2로 컴파일해 열두 입력을 실행합니다. `check`는 소스를 검사하고,
조립 사례와 네이티브 기대값은 `body-compose`가 평가합니다.

`out/retry-first-use/execution.json`을 엽니다.
`runtime.traces[0].deliveries[0].actual`에서 첫 입력
`false, true, 2, 4, 7, 100`의 결과를 확인합니다.

```json
{"retry": true, "delay_ms": 14, "reason": "retry"}
```

입력 여섯 개는 이전 작업 성공 여부, 일시적 실패 여부, 사용한 시도 횟수, 최대 횟수,
직전 대기 시간, 최대 대기 시간입니다. `runtime.finite_passed`와
`runtime.finite_total`은 모두 `12`, `runtime.model_calls`는 `0`이어야 합니다.

| `out/retry-first-use/composition/` 안의 파일 | 읽을 내용 |
| --- | --- |
| `realized.gooo` | 계산한 세 필드를 결과에 연결한 선택된 Gooo 본문 |
| `generated.go` | 해당 본문으로부터 생성된 Go |
| `main.go`, `go.mod` | 네이티브 실행 연결 코드와 Go 모듈 |
| `composition.json` | 다음 명령에서 재사용하는 소스에 연결된 선택 기록 |
| `original.gooo`, `cases.json`, `runtime.json` | 원래 소스, 입력 사례, 네이티브 결과 |

## 4. 저장한 선택 실행

```sh
gooo body-compose --source "$GOOO_RUN/composition/original.gooo" \
  --composition "$GOOO_RUN/composition/composition.json" \
  --cases examples/retry-policy/cases.json > "$GOOO_RUN/replay.json"
```

`replay.json`의 `generated_now`는 `false`, 네이티브 사례는 다시 `12/12`,
`runtime.model_calls`는 `0`이어야 합니다. 새 후보 선택이나 새 모델 호출 없이
저장한 선택을 다시 실행합니다. 재실행에는 네이티브 컴파일과 실행이 포함되며,
새로운 독립 평가 입력이 추가된 것은 아닙니다. 조립부터 다시 실행하려면
`GOOO_RUN`을 `out/retry-first-use-2`처럼 새 경로로 바꿉니다. 기존 출력 폴더는
덮어쓰지 않고 거절합니다.

## 선택 사항: VS Code에서도 같은 컴파일러 사용

[0.2.2 VSIX](https://github.com/kimjooyoon/gooo-ecosystem-workbench/releases/tag/vscode-v0.2.2)를
**Extensions: Install from VSIX...**로 설치하고 `gooo-workbench` 폴더를 엽니다.
VS Code 설정의 `gooo.compilerPath`에 앞서 확인한 `.compiler/bin/gooo`로 끝나는
정확한 절대 경로를 지정합니다. `examples/retry-policy/source.gooo`를 열고
**Gooo: Check Current File**을 실행합니다. 위 터미널 명령으로 예제를 실행한 뒤,
출력 폴더의 `composition/realized.gooo`와 `composition/generated.go`를 엽니다.
새 터미널에서는 같은 `GOTOOLCHAIN`과 절대 PATH 설정이 필요합니다.

호출자가 실패 종류와 시도 횟수를 제공하고, 실제 작업과 기다리기를 수행합니다.
이 정책은 판단을 반환하며 작업 예약이나 API 부수 효과를 수행하지 않습니다.
제공한 유한 사례로 보인 동작을 확인하며 모든 API의 안전성을 증명하지 않습니다.
이 직접 실행 경로에는 Node/npm, 호스팅 모델, Laya 서비스가 필요하지 않습니다.
