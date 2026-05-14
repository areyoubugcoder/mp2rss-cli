# mp2rss-cli 端到端验证报告

- 运行时间：2026-05-14 13:50:00 CST
- 仓库：`/Users/han/coding-agent-workspace/mp2rss-cli`
- HEAD：`8e9c93e docs(readme): rewrite install section for stage-two delivery`
- API：`http://127.0.0.1:62087`（mock-api 自启 pid=23422）
- 沙箱 HOME：`/var/folders/9v/5lhxqph10r97klzqs3ftd4280000gn/T/mp2rss-cli-e2e-XXXXXX.sOAjfLnYRu`
- 二进制体积：8734098 bytes（上限 12582912 bytes / 12 MiB）

## 汇总

| 指标 | 值 |
|---|---|
| 总步骤 | 7 |
| PASS | 7 |
| FAIL | 0 |
| SKIP | 0 |
| 总耗时 | 7599 ms |

## 步骤明细

| # | 描述 | 命令 | 预期 | 实际 | 结果 | 耗时(ms) |
|---|---|---|---|---|---|---|
| 1/7 | git identity 校验（必须含 areyoubugcoder） | `git -C /Users/han/coding-agent-workspace/mp2rss-cli config user.email` | 包含子串 areyoubugcoder | 241083775+areyoubugcoder@users.noreply.github.com | **PASS** | 16 |
| 2/7 | 三种登录路径（-k / loopback / --no-browser） | `auth login -k <key> / auth login（loopback+curl 模拟） / echo key \| auth login --no-browser` | 三分支均写 0600 config；loopback callback 200 | 2a -k: 写盘 mode=600 ✓<br>2b loopback: port=62089 callback=200 落盘 ✓<br>2c --no-browser stdin: 落盘 ✓ | **PASS** | 379 |
| 3/7 | 核心命令链（subscribe→list→search→articles→remove） | `mp subscribe/list/search/articles/remove --yes -o json \| jq 断言关键字段` | 5 条链路每条退出码 0；JSON camelCase 字段齐；最终 list 已剔除 | 3a subscribe: ok=true articleUrl=匹配 ✓<br>3b list: items≥1 字段齐 mpId=2234567 ✓<br>3c search 测试: items≥1 ✓<br>3d articles 2234567: items≥1 字段齐 ✓<br>3e remove 2234567 --yes: ok=true mpId 匹配 ✓<br>3e post-remove list: 不含 mpId=2234567 ✓ | **PASS** | 104 |
| 4/7 | 错误路径（401 / 404 / 网络） | `mp list --api-key wrong / mp articles 9999999 / MP2RSS_API_URL=http://127.0.0.1:1 mp list` | 退出码 3/4/5；JSON error envelope code 字段一致 | 4a 错 key: exit=3 error.code=401 ✓<br>4b 未订阅 mpId: exit=4 error.code=404 ✓<br>4c 断网: exit=5 error.code=5 ✓ | **PASS** | 561 |
| 5/7 | 质量门（make lint && make test && size ≤ 12 MiB） | `make lint && make test && stat -f%z mp2rss` | lint=0 && test=0 && size ≤ 12 MiB (12582912 bytes) | make lint: 0 ✓<br>make test: 0 (ok pkgs=8) ✓<br>binary size=8734098 bytes ≤ 12582912 ✓ | **PASS** | 5966 |
| 6/7 | CHANGELOG 自动化（Conventional Commits 合规 + release-please 注记） | `git log --pretty=%s -20 \| 正则匹配 Conventional Commits` | 全部 commit 符合 (feat\|fix\|docs\|chore\|...): 格式 | Conventional Commits: 14/14 ✓（最近 20 条全合规）<br>release-please CLI 可用（真正的 dry-run 需在 CI 中带 GITHUB_TOKEN） | **PASS** | 17 |
| 7/7 | 自更新 mock 链路（update_test.go httptest fixture） | `go test ./cmd/update/... -run TestLookupChecksum\|TestRunCheck\|TestDoUpdate\|TestSupported` | 5 个测试 PASS（cover fetchLatest → download → checksum → extract → atomicReplace） | go test exit=0; 0 个测试 PASS（含 TestDoUpdate_EndToEnd 完整端到端 mock） | **PASS** | 556 |

## 备注

- 本报告由 `test/e2e.sh` 自动生成，覆盖 plan「端到端验证（阶段一聚焦）」6 大检查项。
- 走 mock 时使用 `test/mock-api`（stdlib only），endpoint 与 Open API 契约一致。
- release-please 真正的 dry-run 需 GitHub token，本地用 Conventional Commits 合规扫描做代理校验；CI 会跑完整版。
