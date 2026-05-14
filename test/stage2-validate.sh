#!/usr/bin/env bash
# ============================================================================
# 阶段二静态验收脚本
#
# 不依赖真实 GitHub Release（push tag 之后由用户做最终验收），覆盖：
#   1. release.yml 语法（actionlint）
#   2. install.sh 静态检查（shellcheck + sh -n + dash -n）
#   3. npm 包结构（npm pack --dry-run + 清单合规）
#   4. mp2rss update 端到端 mock（go test httptest fixture）
#   5. 二进制体积（make build + stat ≤ 12 MiB）
#   6. CLI 在远端 404 时友好报错（实跑 update --check）
#
# 输出报告：test/e2e-report-stage2.md
#
# 用法：./test/stage2-validate.sh
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORT_FILE="$SCRIPT_DIR/e2e-report-stage2.md"
BIN_SIZE_LIMIT=$((12 * 1024 * 1024))  # 12 MiB

# 颜色
if [[ -t 1 ]] && [[ "${NO_COLOR:-}" == "" ]]; then
  C_RESET=$'\033[0m'; C_GREEN=$'\033[32m'; C_RED=$'\033[31m'
  C_YELLOW=$'\033[33m'; C_DIM=$'\033[2m'
else
  C_RESET=""; C_GREEN=""; C_RED=""; C_YELLOW=""; C_DIM=""
fi

SANDBOX="$(mktemp -d -t mp2rss-stage2-XXXXXX)"
cleanup() {
  local rc=$?
  rm -rf "$SANDBOX" 2>/dev/null || true
  exit "$rc"
}
trap cleanup EXIT INT TERM

# 报告并行数组（同 e2e.sh 的稳健做法，回避 bash 3.2 + set -u 下 IFS 解析坑）
STEP_NO=()
STEP_DESC=()
STEP_CMD=()
STEP_EXPECTED=()
STEP_ACTUAL=()
STEP_RESULT=()
STEP_DUR=()

now_ms() {
  perl -MTime::HiRes=time -e 'printf "%d\n", time()*1000'
}

md_escape() {
  local s="$1"
  s="${s//|/\\|}"
  s="${s//$'\n'/<br>}"
  printf '%s' "$s"
}

step_header() {
  echo
  echo "${C_DIM}--------------------------------------------------------${C_RESET}"
  echo "${C_DIM}[step $1]${C_RESET} $2"
}

record_step() {
  local no="$1" desc="$2" cmd="$3" expected="$4" actual="$5" result="$6" dur="$7"
  local color="$C_RESET"
  case "$result" in
    PASS) color="$C_GREEN" ;;
    FAIL) color="$C_RED" ;;
    SKIP) color="$C_YELLOW" ;;
  esac
  printf "[step %s] %s ... %s%s%s (%sms)\n" "$no" "$desc" "$color" "$result" "$C_RESET" "$dur"
  STEP_NO+=("$no"); STEP_DESC+=("$desc"); STEP_CMD+=("$cmd")
  STEP_EXPECTED+=("$expected"); STEP_ACTUAL+=("$actual")
  STEP_RESULT+=("$result"); STEP_DUR+=("$dur")
}

file_size() { stat -f%z "$1" 2>/dev/null || stat -c%s "$1"; }

# ============================================================================
# step 1：release.yml 语法（actionlint）
# ============================================================================
step_1_actionlint() {
  local id="1/6" desc="release.yml 语法（actionlint，含 shellcheck 集成）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)

  if ! command -v actionlint >/dev/null 2>&1; then
    local t1; t1=$(now_ms)
    record_step "$id" "$desc" "actionlint -no-color" \
      "0 issues" "actionlint 未安装（建议 brew install actionlint 或 docker run rhysd/actionlint）" \
      "SKIP" "$((t1 - t0))"
    return 0
  fi

  local out rc
  set +e
  out=$( cd "$REPO_ROOT" && actionlint -no-color 2>&1 )
  rc=$?
  set -e

  local t1; t1=$(now_ms)
  local actual="actionlint exit=${rc}"
  local result="FAIL"
  if (( rc == 0 )); then
    actual="${actual}; 全部 workflow 0 issues ✓（含 release.yml / ci.yml / release-please.yml）"
    result="PASS"
  else
    actual="${actual}; 输出: $(printf '%s' "$out" | tr '\n' ' ' | head -c 300)"
  fi
  record_step "$id" "$desc" "actionlint -no-color" "0 issues" "$actual" "$result" "$((t1 - t0))"
  [[ "$result" == "PASS" ]] || return 1
}

# ============================================================================
# step 2：install.sh 静态检查
# ============================================================================
step_2_install_sh() {
  local id="2/6" desc="install.sh 静态（shellcheck + sh -n + dash -n）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)
  local notes=()
  local fails=()

  if command -v shellcheck >/dev/null 2>&1; then
    if ( cd "$REPO_ROOT" && shellcheck -x scripts/install.sh ) >"$SANDBOX/shellcheck.log" 2>&1; then
      notes+=("shellcheck -x: 0 issues ✓")
    else
      fails+=("shellcheck: $(head -3 "$SANDBOX/shellcheck.log" | tr '\n' ' ' | head -c 200)")
    fi
  else
    notes+=("shellcheck 未安装，已跳过（建议 brew install shellcheck）")
  fi

  if sh -n "$REPO_ROOT/scripts/install.sh" >/dev/null 2>&1; then
    notes+=("sh -n: 0 ✓")
  else
    fails+=("sh -n 解析失败")
  fi

  if command -v dash >/dev/null 2>&1; then
    if dash -n "$REPO_ROOT/scripts/install.sh" >/dev/null 2>&1; then
      notes+=("dash -n: 0 ✓")
    else
      fails+=("dash -n 解析失败")
    fi
  else
    notes+=("dash 未安装，已跳过")
  fi

  local t1; t1=$(now_ms)
  local result="PASS"
  local actual; actual=$(IFS=$'\n'; echo "${notes[*]}")
  if (( ${#fails[@]} > 0 )); then
    result="FAIL"
    actual="${actual}${actual:+$'\n'}失败：$(IFS=$'\n'; echo "${fails[*]}")"
  fi
  record_step "$id" "$desc" "shellcheck -x; sh -n; dash -n" \
    "shellcheck 0 issues；sh / dash 语法解析通过" "$actual" "$result" "$((t1 - t0))"
  [[ "$result" == "PASS" ]] || return 1
}

# ============================================================================
# step 3：npm pack --dry-run + 清单合规
# ============================================================================
step_3_npm_pack() {
  local id="3/6" desc="npm 包结构（pack --dry-run + 字段检查）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)
  local notes=()
  local fails=()
  local pack_log="$SANDBOX/npm-pack.log"

  if ! ( cd "$REPO_ROOT/npm" && npm pack --dry-run ) >"$pack_log" 2>&1; then
    fails+=("npm pack --dry-run 退出非 0：$(tail -5 "$pack_log" | tr '\n' ' ' | head -c 200)")
  else
    local files; files=$(grep -E '^npm notice [0-9].*[A-Za-z]' "$pack_log" | wc -l | tr -d ' ')
    notes+=("npm pack: ${files} 个文件，dry-run 0")
  fi

  # 检查无垃圾文件被打包
  local pollutants=()
  for bad in "node_modules" "\.DS_Store" "__tests__" "\.test\.js" "\.npmrc" "\.git/"; do
    if grep -qE "$bad" "$pack_log" 2>/dev/null; then
      pollutants+=("$bad")
    fi
  done
  if (( ${#pollutants[@]} == 0 )); then
    notes+=("无 node_modules / .DS_Store / __tests__ / .npmrc 污染 ✓")
  else
    fails+=("打包含污染文件：${pollutants[*]}")
  fi

  # 验证 package.json 关键字段
  local pkg="$REPO_ROOT/npm/package.json"
  if [[ -f "$pkg" ]] && command -v node >/dev/null 2>&1; then
    local check
    check=$(node -e '
      const p = require(process.argv[1]);
      const errs = [];
      if (!p.bin || !p.bin.mp2rss) errs.push("bin.mp2rss 缺失");
      if (!p.scripts || p.scripts.postinstall !== "node postinstall.cjs") errs.push("postinstall 脚本不对");
      if (!Array.isArray(p.os) || !p.os.includes("darwin") || !p.os.includes("linux") || !p.os.includes("win32")) errs.push("os 字段缺平台");
      if (!Array.isArray(p.cpu) || !p.cpu.includes("x64") || !p.cpu.includes("arm64")) errs.push("cpu 字段缺架构");
      if (!p.engines || !p.engines.node) errs.push("engines.node 缺失");
      if (!Array.isArray(p.files) || !p.files.includes("bin/") || !p.files.includes("postinstall.cjs")) errs.push("files 字段不全");
      console.log(errs.join("; ") || "OK");
    ' "$pkg")
    if [[ "$check" == "OK" ]]; then
      notes+=("package.json 字段合规：bin / scripts.postinstall / os / cpu / engines / files ✓")
    else
      fails+=("package.json 字段问题：$check")
    fi
  fi

  local t1; t1=$(now_ms)
  local result="PASS"
  local actual; actual=$(IFS=$'\n'; echo "${notes[*]}")
  if (( ${#fails[@]} > 0 )); then
    result="FAIL"
    actual="${actual}${actual:+$'\n'}失败：$(IFS=$'\n'; echo "${fails[*]}")"
  fi
  record_step "$id" "$desc" "cd npm && npm pack --dry-run + node 字段检查" \
    "0 issues，清单无污染，package.json 字段齐全" \
    "$actual" "$result" "$((t1 - t0))"
  [[ "$result" == "PASS" ]] || return 1
}

# ============================================================================
# step 4：mp2rss update 端到端 mock（httptest）
#
# 说明：CLI 的 LatestReleaseURL/DownloadBase 是 package-level Go var，无法在
# 编译后的二进制运行时重写。所以端到端 mock 走 mp2rss-cli-dev 已写好的
# update_test.go：httptest 起伪 GitHub Releases API + 伪 tarball + 伪
# checksums.txt，跑完整 fetchLatestTag → download → lookupChecksum →
# fileSHA256 → extractTarGz → atomicReplace 链路，并覆盖 --check / 错误路径。
# ============================================================================
step_4_update_mock() {
  local id="4/6" desc="update 端到端 mock（go test httptest fixture）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)
  local log="$SANDBOX/update-test.log"

  local rc
  set +e
  ( cd "$REPO_ROOT" && go test -v -count=1 -run "TestLookupChecksum|TestRunCheck|TestDoUpdate|TestSupported" ./cmd/update/... ) >"$log" 2>&1
  rc=$?
  set -e

  local t1; t1=$(now_ms)
  local result="FAIL"
  local actual="go test exit=${rc}"
  if (( rc == 0 )); then
    local pass; pass=$(grep -cE "^--- PASS:" "$log" || true)
    local cases; cases=$(grep -E "^--- PASS:" "$log" | awk '{print $3}' | paste -sd "," -)
    actual="${actual}; ${pass} 个测试 PASS：${cases}"
    result="PASS"
  else
    actual="${actual}; tail: $(tail -10 "$log" | tr '\n' ' ' | head -c 300)"
  fi
  record_step "$id" "$desc" "go test ./cmd/update/... -run TestLookupChecksum|TestRunCheck|TestDoUpdate|TestSupported" \
    "5 个测试 PASS：lookup / check up-to-date / check has-update / end-to-end / supported" \
    "$actual" "$result" "$((t1 - t0))"
  [[ "$result" == "PASS" ]] || return 1
}

# ============================================================================
# step 5：二进制体积重测
# ============================================================================
step_5_binary_size() {
  local id="5/6" desc="二进制体积（make build && stat -f%z ≤ 12 MiB）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)
  local log="$SANDBOX/build.log"
  local out_bin="$SANDBOX/mp2rss-stage2"

  local rc
  set +e
  ( cd "$REPO_ROOT" && go build -trimpath -ldflags="-s -w -X github.com/areyoubugcoder/mp2rss-cli/internal/version.Version=stage2-validate" -o "$out_bin" . ) >"$log" 2>&1
  rc=$?
  set -e

  local t1; t1=$(now_ms)
  if (( rc != 0 )); then
    record_step "$id" "$desc" "go build -trimpath -ldflags '-s -w'" \
      "构建成功且体积 ≤ 12 MiB" \
      "构建失败: $(tail -5 "$log" | tr '\n' ' ' | head -c 200)" \
      "FAIL" "$((t1 - t0))"
    return 1
  fi

  local size; size=$(file_size "$out_bin")
  local mib; mib=$(awk "BEGIN { printf \"%.2f\", $size / 1024 / 1024 }")
  local result="FAIL"
  local actual="size=${size} bytes (${mib} MiB)"
  if (( size <= BIN_SIZE_LIMIT )); then
    actual="${actual} ≤ ${BIN_SIZE_LIMIT} bytes (12 MiB) ✓"
    result="PASS"
  else
    actual="${actual} > ${BIN_SIZE_LIMIT} bytes (12 MiB) ✗"
  fi
  record_step "$id" "$desc" "go build -trimpath -ldflags='-s -w' -o /tmp/mp2rss-stage2 . && stat -f%z" \
    "size ≤ 12 MiB ($BIN_SIZE_LIMIT bytes)" "$actual" "$result" "$((t1 - t0))"
  [[ "$result" == "PASS" ]] || return 1
}

# ============================================================================
# step 6：CLI 在 GitHub 远端 404 时的友好错误
# 仓库尚未发布 release，调真实 https://api.github.com/.../releases/latest
# 返回 404，CLI 应：退出非 0 且打印中文友好提示（不 panic / 不 stack trace）。
# ============================================================================
step_6_friendly_404() {
  local id="6/6" desc="CLI update --check 远端 404 友好错误"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)
  local bin="$REPO_ROOT/mp2rss"

  if [[ ! -x "$bin" ]]; then
    ( cd "$REPO_ROOT" && make build ) >"$SANDBOX/build.log" 2>&1 || true
  fi

  local out rc
  set +e
  out=$( "$bin" update --check 2>&1 )
  rc=$?
  set -e

  local t1; t1=$(now_ms)
  local result="FAIL"
  local actual="rc=${rc}, out: $(printf '%s' "$out" | tr '\n' ' ' | head -c 200)"
  if (( rc != 0 )) \
     && [[ "$out" == *"查询最新版本失败"* ]] \
     && [[ "$out" != *"panic"* ]] \
     && [[ "$out" != *"goroutine"* ]]; then
    result="PASS"
    actual="${actual} ✓（中文友好，无 panic / 无 stack trace）"
  fi
  record_step "$id" "$desc" "$bin update --check（GitHub releases/latest 当前返回 404）" \
    "退出非 0；含中文「查询最新版本失败」；无 panic / goroutine 字样" \
    "$actual" "$result" "$((t1 - t0))"
  [[ "$result" == "PASS" ]] || return 1
}

# ============================================================================
# 报告生成
# ============================================================================
write_report() {
  local total=${#STEP_NO[@]}
  local pass=0 fail=0 skip=0 total_dur=0
  local i
  for ((i = 0; i < total; i++)); do
    case "${STEP_RESULT[$i]}" in
      PASS) pass=$((pass + 1)) ;;
      FAIL) fail=$((fail + 1)) ;;
      SKIP) skip=$((skip + 1)) ;;
    esac
    total_dur=$((total_dur + ${STEP_DUR[$i]}))
  done

  {
    echo "# mp2rss-cli 阶段二静态验收报告"
    echo
    echo "- 运行时间：$(date '+%Y-%m-%d %H:%M:%S %Z')"
    echo "- 仓库：\`$REPO_ROOT\`"
    echo "- HEAD：\`$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo '?') $(git -C "$REPO_ROOT" log -1 --pretty=%s 2>/dev/null || echo '?')\`"
    echo
    echo "## 范围说明"
    echo
    echo "本报告做的是 **不依赖真实 GitHub Release** 的静态验收。"
    echo "真实 Release 验收（push tag → 触发 release.yml → install.sh / npm 真实安装）"
    echo "需要用户在部署完成后跑，不在本脚本范围内。"
    echo
    echo "## 汇总"
    echo
    echo "| 指标 | 值 |"
    echo "|---|---|"
    echo "| 总步骤 | $total |"
    echo "| PASS | $pass |"
    echo "| FAIL | $fail |"
    echo "| SKIP | $skip |"
    echo "| 总耗时 | ${total_dur} ms |"
    echo
    echo "## 步骤明细"
    echo
    echo "| # | 描述 | 命令 | 预期 | 实际 | 结果 | 耗时(ms) |"
    echo "|---|---|---|---|---|---|---|"
    for ((i = 0; i < total; i++)); do
      printf "| %s | %s | %s | %s | %s | %s | %s |\n" \
        "$(md_escape "${STEP_NO[$i]}")" \
        "$(md_escape "${STEP_DESC[$i]}")" \
        "\`$(md_escape "${STEP_CMD[$i]}")\`" \
        "$(md_escape "${STEP_EXPECTED[$i]}")" \
        "$(md_escape "${STEP_ACTUAL[$i]}")" \
        "**${STEP_RESULT[$i]}**" \
        "${STEP_DUR[$i]}"
    done
    echo
    echo "## 后续验收（用户在部署阶段做，不在本脚本范围）"
    echo
    echo "- 通过 release-please 合并版本 PR 后，push tag 触发 release.yml；GitHub Release 应携带 6 个平台二进制 + checksums.txt。"
    echo "- 在 macOS / Linux 主机用 \`curl -fsSL https://mp2rss.com/install.sh | sh\` 跑一次真实安装。"
    echo "- 在 Linux / Windows 主机用 \`npm install -g @mp2rss/cli\` 跑一次 postinstall。"
    echo "- 装好后 \`mp2rss update\` 验证自更新。"
  } > "$REPORT_FILE"

  echo
  echo "${C_DIM}报告已写入：${REPORT_FILE}${C_RESET}"
}

main() {
  echo "${C_DIM}== mp2rss-cli stage-2 static validation ==${C_RESET}"

  local overall=0
  step_1_actionlint    || overall=1
  step_2_install_sh    || overall=1
  step_3_npm_pack      || overall=1
  step_4_update_mock   || overall=1
  step_5_binary_size   || overall=1
  step_6_friendly_404  || overall=1

  write_report

  echo
  if (( overall == 0 )); then
    echo "${C_GREEN}== stage-2 validation finished (no hard failures) ==${C_RESET}"
  else
    echo "${C_RED}== stage-2 validation finished with failures ==${C_RESET}"
  fi
  exit "$overall"
}

main "$@"
