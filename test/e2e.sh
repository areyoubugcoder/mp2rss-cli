#!/usr/bin/env bash
# ============================================================================
# mp2rss-cli 端到端验证脚本（阶段一）
#
# 状态：骨架（scaffold）。Tasks #2/#3 完成后再把各 step 内的 TODO 替换为真实
# CLI 命令。当前所有 step 都用 SKIP 占位，脚本本身可完整跑通，验证基础设施
# （环境变量解析、临时目录、mock-api 自启、报告生成）是否正常。
#
# 用法：
#   ./test/e2e.sh
#
# 环境变量：
#   MP2RSS_DEV_API_URL       默认 http://localhost:3001。若设了 MP2RSS_TEST_FEED_KEY
#                            就用此真实 API，否则自动启 mock。
#   MP2RSS_TEST_FEED_KEY     可选。设置后走真实 API；未设置则脚本启 mock，
#                            内部用固定 key（见 test/mock-api/main.go）。
#   MP2RSS_TEST_ARTICLE_URL  可选。subscribe 用的文章 URL。
#                            未设置且走 mock 时使用默认值。
# ============================================================================
set -euo pipefail

# ---------------- 路径与常量 ----------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORT_FILE="$SCRIPT_DIR/e2e-report.md"
MOCK_FEED_KEY="test-feed-key-0123456789abcdef"
MOCK_ARTICLE_URL_DEFAULT="https://mp.weixin.qq.com/s/test-article-001"
BIN_SIZE_LIMIT=$((12 * 1024 * 1024))  # 12 MiB

# ---------------- 颜色 / 输出 ----------------
if [[ -t 1 ]] && [[ "${NO_COLOR:-}" == "" ]]; then
  C_RESET=$'\033[0m'; C_GREEN=$'\033[32m'; C_RED=$'\033[31m'
  C_YELLOW=$'\033[33m'; C_DIM=$'\033[2m'
else
  C_RESET=""; C_GREEN=""; C_RED=""; C_YELLOW=""; C_DIM=""
fi

# ---------------- 临时环境隔离 ----------------
SANDBOX_HOME="$(mktemp -d -t mp2rss-cli-e2e-XXXXXX)"
MOCK_PID=""
MOCK_PORT=""
MOCK_LOG="$SANDBOX_HOME/mock-api.log"

cleanup() {
  local exit_code=$?
  if [[ -n "$MOCK_PID" ]]; then
    kill "$MOCK_PID" 2>/dev/null || true
    wait "$MOCK_PID" 2>/dev/null || true
  fi
  rm -rf "$SANDBOX_HOME" 2>/dev/null || true
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

# ---------------- 报告数据结构 ----------------
# 每个 step 完成后追加一行到 STEPS 数组：
#   "<no>|<desc>|<cmd>|<expected>|<actual>|<result>|<duration_ms>"
STEPS=()

now_ms() {
  # macOS 的 date 不支持 %N，所以用 perl
  perl -MTime::HiRes=time -e 'printf "%d\n", time()*1000'
}

# 打印 step 头
step_header() {
  local id="$1" desc="$2"
  echo
  echo "${C_DIM}--------------------------------------------------------${C_RESET}"
  echo "${C_DIM}[step $id]${C_RESET} $desc"
}

# 记录 step 结果到数组并打印
# 参数：no desc cmd expected actual result duration_ms
record_step() {
  local no="$1" desc="$2" cmd="$3" expected="$4" actual="$5" result="$6" dur="$7"
  local color="$C_RESET"
  case "$result" in
    PASS) color="$C_GREEN" ;;
    FAIL) color="$C_RED" ;;
    SKIP) color="$C_YELLOW" ;;
  esac
  printf "[step %s] %s ... %s%s%s (%sms)\n" \
    "$no" "$desc" "$color" "$result" "$C_RESET" "$dur"
  STEPS+=("${no}|${desc}|${cmd}|${expected}|${actual}|${result}|${dur}")
}

# 用 escape 表格管道字符
md_escape() {
  # 替换 | -> \|，换行 -> <br>
  local s="$1"
  s="${s//|/\\|}"
  s="${s//$'\n'/<br>}"
  printf '%s' "$s"
}

# ---------------- Mock API 启动 ----------------
maybe_start_mock() {
  if [[ -n "${MP2RSS_TEST_FEED_KEY:-}" ]]; then
    echo "${C_DIM}使用真实 API：${MP2RSS_DEV_API_URL:-http://localhost:3001}${C_RESET}"
    API_URL="${MP2RSS_DEV_API_URL:-http://localhost:3001}"
    FEED_KEY="$MP2RSS_TEST_FEED_KEY"
    ARTICLE_URL="${MP2RSS_TEST_ARTICLE_URL:-}"
    if [[ -z "$ARTICLE_URL" ]]; then
      echo "${C_RED}错误：使用真实 API 时必须设置 MP2RSS_TEST_ARTICLE_URL${C_RESET}" >&2
      exit 2
    fi
    return
  fi

  echo "${C_DIM}未提供 MP2RSS_TEST_FEED_KEY，自动启动 mock-api...${C_RESET}"
  # 先编译成单一二进制，避免 `go run` 双层进程导致 trap 杀不干净。
  local mock_bin="$SANDBOX_HOME/mock-api"
  ( cd "$REPO_ROOT" && go build -o "$mock_bin" ./test/mock-api/ ) \
    || { echo "${C_RED}错误：mock-api 编译失败${C_RESET}" >&2; exit 1; }
  "$mock_bin" -port 0 >"$MOCK_LOG" 2>&1 &
  MOCK_PID=$!

  # 等待 stdout 第一行 "listening on :NNNN"
  local waited=0
  while (( waited < 100 )); do
    if [[ -s "$MOCK_LOG" ]] && grep -q "^listening on :" "$MOCK_LOG"; then
      break
    fi
    sleep 0.1
    waited=$((waited + 1))
  done

  if ! grep -q "^listening on :" "$MOCK_LOG"; then
    echo "${C_RED}错误：mock-api 启动失败，日志：${C_RESET}" >&2
    cat "$MOCK_LOG" >&2 || true
    exit 1
  fi

  MOCK_PORT=$(grep "^listening on :" "$MOCK_LOG" | head -1 | sed 's/^listening on ://')
  API_URL="http://127.0.0.1:${MOCK_PORT}"
  FEED_KEY="$MOCK_FEED_KEY"
  ARTICLE_URL="${MP2RSS_TEST_ARTICLE_URL:-$MOCK_ARTICLE_URL_DEFAULT}"
  echo "${C_DIM}mock-api 已启动：$API_URL (pid=$MOCK_PID)${C_RESET}"
}

# ============================================================================
# 6 大步骤
# ============================================================================

# ---- step 1：git identity 校验 -----------------------------------
step_1_git_identity() {
  local id="1/6" desc="git identity 校验（必须含 areyoubugcoder）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)
  local cmd="git -C $REPO_ROOT config user.email"
  local actual; actual=$(git -C "$REPO_ROOT" config user.email 2>&1 || echo "")
  local expected="包含子串 areyoubugcoder"
  local result="FAIL"
  if [[ "$actual" == *"areyoubugcoder"* ]]; then
    result="PASS"
  fi
  local t1; t1=$(now_ms); local dur=$((t1 - t0))
  record_step "$id" "$desc" "$cmd" "$expected" "$actual" "$result" "$dur"
  [[ "$result" == "PASS" ]] || return 1
}

# ---- step 2：三种登录路径 ----------------------------------------
step_2_login_flows() {
  local id="2/6" desc="三种登录路径（-k / loopback / --no-browser）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)

  # TODO: wire up after #2/#3 land
  #   2a) mp2rss auth login -k "$FEED_KEY" --api-url "$API_URL"
  #       预期：写入 $SANDBOX_HOME/.mp2rss/config.json，权限 0600
  #   2b) loopback：mp2rss auth login --api-url "$API_URL" --port 28080
  #       预期：等待 callback；脚本端 fetch 127.0.0.1:28080/cli/callback 模拟前端 POST
  #            (or expect-based 半自动)
  #   2c) --no-browser：mp2rss auth login --no-browser --api-url "$API_URL"
  #       预期：打印授权 URL；可粘贴 key 走备用通道

  local cmd="(TODO: 三个登录子路径)"
  local expected="2a 写盘 0600；2b loopback callback 命中；2c 备用通道落盘"
  local actual="skeleton: CLI 命令尚未接入"
  local result="SKIP"

  local t1; t1=$(now_ms); local dur=$((t1 - t0))
  record_step "$id" "$desc" "$cmd" "$expected" "$actual" "$result" "$dur"
}

# ---- step 3：核心命令 ------------------------------------------------
step_3_core_commands() {
  local id="3/6" desc="核心命令链路（subscribe→list→search→articles→remove）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)

  # TODO: wire up after #2/#3 land
  #   3a) mp2rss mp subscribe "$ARTICLE_URL" -o json | jq -e '.ok == true'
  #   3b) mp2rss mp list -o json | jq -e '.items | length >= 1'
  #       记下首个 mpId 到 MP_ID
  #   3c) mp2rss mp search "测试" -o json | jq -e '.items | length >= 1'
  #   3d) mp2rss mp articles "$MP_ID" -o json | jq -e '.items | length >= 1'
  #   3e) mp2rss mp remove "$MP_ID" --yes -o json | jq -e '.ok == true'
  #       后再 list 应不含该 mpId

  local cmd="(TODO: subscribe/list/search/articles/remove 五连)"
  local expected="每步退出码 0；JSON 关键字段断言通过"
  local actual="skeleton: CLI 命令尚未接入"
  local result="SKIP"

  local t1; t1=$(now_ms); local dur=$((t1 - t0))
  record_step "$id" "$desc" "$cmd" "$expected" "$actual" "$result" "$dur"
}

# ---- step 4：错误路径 ----------------------------------------------
step_4_error_paths() {
  local id="4/6" desc="错误路径（错 key / 未订阅 mpId / 断网）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)

  # TODO: wire up after #2/#3 land
  #   4a) MP2RSS_FEED_KEY=wrong-key mp2rss mp list --api-url "$API_URL"
  #       预期：退出码 3（鉴权失败），stderr 含 401
  #   4b) mp2rss mp articles 9999999 --api-url "$API_URL"
  #       预期：退出码 4（资源不存在），stderr 含 "未订阅" / "not subscribed"
  #   4c) MP2RSS_API_URL=http://127.0.0.1:1 mp2rss mp list
  #       预期：退出码 5（上游不可用）或 1（通用），不 panic，stderr 中文友好提示

  local cmd="(TODO: 错 key / 未订阅 / 断网 三组)"
  local expected="退出码分别为 3 / 4 / 5（或 1）；不 panic"
  local actual="skeleton: CLI 命令尚未接入"
  local result="SKIP"

  local t1; t1=$(now_ms); local dur=$((t1 - t0))
  record_step "$id" "$desc" "$cmd" "$expected" "$actual" "$result" "$dur"
}

# ---- step 5：质量门（lint + test + binary size） -----------------
step_5_quality_gates() {
  local id="5/6" desc="质量门（make lint && make test && build size ≤ 12MiB）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)

  # TODO: wire up after #4 lands Makefile：
  #   make -C "$REPO_ROOT" lint
  #   make -C "$REPO_ROOT" test
  #   make -C "$REPO_ROOT" build
  #   size=$(stat -f%z "$REPO_ROOT/bin/mp2rss" 2>/dev/null || stat -c%s "$REPO_ROOT/bin/mp2rss")
  #   (( size <= BIN_SIZE_LIMIT ))

  local cmd="make lint && make test && make build (TODO)"
  local expected="lint=0 && test=0 && binary size ≤ $BIN_SIZE_LIMIT bytes"
  local actual="skeleton: Makefile 尚未接入（依赖 #4）"
  local result="SKIP"

  local t1; t1=$(now_ms); local dur=$((t1 - t0))
  record_step "$id" "$desc" "$cmd" "$expected" "$actual" "$result" "$dur"
}

# ---- step 6：CHANGELOG dry-run -----------------------------------
step_6_changelog_dryrun() {
  local id="6/6" desc="CHANGELOG 自动化 dry-run（release-please）"
  step_header "$id" "$desc"
  local t0; t0=$(now_ms)

  # TODO: wire up after #4 lands release-please-config.json：
  #   npx --yes release-please \
  #     --token=dummy \
  #     --repo-url=https://github.com/areyoubugcoder/mp2rss-cli \
  #     --dry-run release-pr 2>&1 | tee "$SANDBOX_HOME/release-please.log"
  #   解析 stdout 列出会生成的 PR 标题

  local cmd="npx release-please --dry-run release-pr (TODO)"
  local expected="dry-run 列出至少一条 release PR 标题，或'no release'"
  local actual="skeleton: release-please 配置尚未接入（依赖 #4）"
  local result="SKIP"

  local t1; t1=$(now_ms); local dur=$((t1 - t0))
  record_step "$id" "$desc" "$cmd" "$expected" "$actual" "$result" "$dur"
}

# ============================================================================
# 报告生成
# ============================================================================
write_report() {
  local total=${#STEPS[@]}
  local pass=0 fail=0 skip=0 total_dur=0
  for line in "${STEPS[@]}"; do
    IFS='|' read -r _ _ _ _ _ r d <<<"$line"
    case "$r" in
      PASS) pass=$((pass + 1)) ;;
      FAIL) fail=$((fail + 1)) ;;
      SKIP) skip=$((skip + 1)) ;;
    esac
    total_dur=$((total_dur + d))
  done

  {
    echo "# mp2rss-cli 端到端验证报告"
    echo
    echo "- 运行时间：$(date '+%Y-%m-%d %H:%M:%S %Z')"
    echo "- 仓库：\`$REPO_ROOT\`"
    echo "- API：\`${API_URL:-?}\`（$([ -n "$MOCK_PID" ] && echo "mock，pid=$MOCK_PID" || echo "真实 dev")）"
    echo "- 沙箱 HOME：\`$SANDBOX_HOME\`"
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
    for line in "${STEPS[@]}"; do
      IFS='|' read -r no desc cmd expected actual r d <<<"$line"
      printf "| %s | %s | %s | %s | %s | %s | %s |\n" \
        "$(md_escape "$no")" \
        "$(md_escape "$desc")" \
        "\`$(md_escape "$cmd")\`" \
        "$(md_escape "$expected")" \
        "$(md_escape "$actual")" \
        "**$r**" \
        "$d"
    done
    echo
    echo "## 备注"
    echo
    echo "本报告由 \`test/e2e.sh\` 自动生成。骨架阶段所有 step 均为 SKIP，"
    echo "Tasks #2–#7 完成后会逐个把 TODO 替换为真实 CLI 命令并产出 PASS/FAIL。"
  } > "$REPORT_FILE"

  echo
  echo "${C_DIM}报告已写入：${REPORT_FILE}${C_RESET}"
}

# ============================================================================
# 主流程
# ============================================================================
main() {
  echo "${C_DIM}== mp2rss-cli e2e harness ==${C_RESET}"
  echo "${C_DIM}沙箱 HOME：$SANDBOX_HOME${C_RESET}"

  # CLI 配置写到沙箱 HOME，避免污染本机 ~/.mp2rss
  export HOME="$SANDBOX_HOME"

  maybe_start_mock

  local overall=0
  step_1_git_identity || overall=1
  step_2_login_flows  || overall=1
  step_3_core_commands || overall=1
  step_4_error_paths   || overall=1
  step_5_quality_gates || overall=1
  step_6_changelog_dryrun || overall=1

  write_report

  echo
  if (( overall == 0 )); then
    echo "${C_GREEN}== e2e harness finished (no hard failures) ==${C_RESET}"
  else
    echo "${C_RED}== e2e harness finished with failures ==${C_RESET}"
  fi
  exit "$overall"
}

main "$@"
