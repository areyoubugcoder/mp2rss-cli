# mp2rss-cli 阶段二静态验收报告

- 运行时间：2026-05-14 13:47:52 CST
- 仓库：`/Users/han/coding-agent-workspace/mp2rss-cli`
- HEAD：`8e9c93e docs(readme): rewrite install section for stage-two delivery`

## 范围说明

本报告做的是 **不依赖真实 GitHub Release** 的静态验收。
真实 Release 验收（push tag → 触发 release.yml → install.sh / npm 真实安装）
需要用户在部署完成后跑，不在本脚本范围内。

## 汇总

| 指标 | 值 |
|---|---|
| 总步骤 | 6 |
| PASS | 6 |
| FAIL | 0 |
| SKIP | 0 |
| 总耗时 | 3153 ms |

## 步骤明细

| # | 描述 | 命令 | 预期 | 实际 | 结果 | 耗时(ms) |
|---|---|---|---|---|---|---|
| 1/6 | release.yml 语法（actionlint，含 shellcheck 集成） | `actionlint -no-color` | 0 issues | actionlint exit=0; 全部 workflow 0 issues ✓（含 release.yml / ci.yml / release-please.yml） | **PASS** | 153 |
| 2/6 | install.sh 静态（shellcheck + sh -n + dash -n） | `shellcheck -x; sh -n; dash -n` | shellcheck 0 issues；sh / dash 语法解析通过 | shellcheck -x: 0 issues ✓<br>sh -n: 0 ✓<br>dash -n: 0 ✓ | **PASS** | 100 |
| 3/6 | npm 包结构（pack --dry-run + 字段检查） | `cd npm && npm pack --dry-run + node 字段检查` | 0 issues，清单无污染，package.json 字段齐全 | npm pack: 4 个文件，dry-run 0<br>无 node_modules / .DS_Store / __tests__ / .npmrc 污染 ✓<br>package.json 字段合规：bin / scripts.postinstall / os / cpu / engines / files ✓ | **PASS** | 477 |
| 4/6 | update 端到端 mock（go test httptest fixture） | `go test ./cmd/update/... -run TestLookupChecksum\|TestRunCheck\|TestDoUpdate\|TestSupported` | 5 个测试 PASS：lookup / check up-to-date / check has-update / end-to-end / supported | go test exit=0; 5 个测试 PASS：TestLookupChecksum,TestRunCheck_UpToDate,TestRunCheck_HasUpdate,TestDoUpdate_EndToEnd,TestSupported | **PASS** | 1012 |
| 5/6 | 二进制体积（make build && stat -f%z ≤ 12 MiB） | `go build -trimpath -ldflags='-s -w' -o /tmp/mp2rss-stage2 . && stat -f%z` | size ≤ 12 MiB (12582912 bytes) | size=8734098 bytes (8.33 MiB) ≤ 12582912 bytes (12 MiB) ✓ | **PASS** | 689 |
| 6/6 | CLI update --check 远端 404 友好错误 | `/Users/han/coding-agent-workspace/mp2rss-cli/mp2rss update --check（GitHub releases/latest 当前返回 404）` | 退出非 0；含中文「查询最新版本失败」；无 panic / goroutine 字样 | rc=1, out: 🔎 正在查询最新版本… ✗ 查询最新版本失败：GitHub API HTTP 404 ✓（中文友好，无 panic / 无 stack trace） | **PASS** | 722 |

## 后续验收（用户在部署阶段做，不在本脚本范围）

- 通过 release-please 合并版本 PR 后，push tag 触发 release.yml；GitHub Release 应携带 6 个平台二进制 + checksums.txt。
- 在 macOS / Linux 主机用 `curl -fsSL https://raw.githubusercontent.com/areyoubugcoder/mp2rss-cli/main/scripts/install.sh | sh` 跑一次真实安装。
- 在 Linux / Windows 主机用 `npm install -g @mp2rss/cli` 跑一次 postinstall。
- 装好后 `mp2rss update` 验证自更新。
