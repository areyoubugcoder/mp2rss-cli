# mp2rss-cli 端到端验证报告

- 运行时间：2026-05-14 11:42:27 CST
- 仓库：`/Users/han/coding-agent-workspace/mp2rss-cli`
- API：`http://127.0.0.1:56381`（mock，pid=77017）
- 沙箱 HOME：`/var/folders/9v/5lhxqph10r97klzqs3ftd4280000gn/T/mp2rss-cli-e2e-XXXXXX.t5UYFLihsK`

## 汇总

| 指标 | 值 |
|---|---|
| 总步骤 | 6 |
| PASS | 1 |
| FAIL | 0 |
| SKIP | 5 |
| 总耗时 | 72 ms |

## 步骤明细

| # | 描述 | 命令 | 预期 | 实际 | 结果 | 耗时(ms) |
|---|---|---|---|---|---|---|
| 1/6 | git identity 校验（必须含 areyoubugcoder） | `git -C /Users/han/coding-agent-workspace/mp2rss-cli config user.email` | 包含子串 areyoubugcoder | 241083775+areyoubugcoder@users.noreply.github.com | **PASS** | 16 |
| 2/6 | 三种登录路径（-k / loopback / --no-browser） | `(TODO: 三个登录子路径)` | 2a 写盘 0600；2b loopback callback 命中；2c 备用通道落盘 | skeleton: CLI 命令尚未接入 | **SKIP** | 8 |
| 3/6 | 核心命令链路（subscribe→list→search→articles→remove） | `(TODO: subscribe/list/search/articles/remove 五连)` | 每步退出码 0；JSON 关键字段断言通过 | skeleton: CLI 命令尚未接入 | **SKIP** | 10 |
| 4/6 | 错误路径（错 key / 未订阅 mpId / 断网） | `(TODO: 错 key / 未订阅 / 断网 三组)` | 退出码分别为 3 / 4 / 5（或 1）；不 panic | skeleton: CLI 命令尚未接入 | **SKIP** | 20 |
| 5/6 | 质量门（make lint && make test && build size ≤ 12MiB） | `make lint && make test && make build (TODO)` | lint=0 && test=0 && binary size ≤ 12582912 bytes | skeleton: Makefile 尚未接入（依赖 #4） | **SKIP** | 9 |
| 6/6 | CHANGELOG 自动化 dry-run（release-please） | `npx release-please --dry-run release-pr (TODO)` | dry-run 列出至少一条 release PR 标题，或'no release' | skeleton: release-please 配置尚未接入（依赖 #4） | **SKIP** | 9 |

## 备注

本报告由 `test/e2e.sh` 自动生成。骨架阶段所有 step 均为 SKIP，
Tasks #2–#7 完成后会逐个把 TODO 替换为真实 CLI 命令并产出 PASS/FAIL。
