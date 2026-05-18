# test/ — 端到端验证脚手架

本目录仅用于 **本地端到端回归**，不参与 CLI 主二进制的构建、发布或运行时。

## 目录速览

| 路径 | 用途 |
|---|---|
| `e2e.sh` | bash 端到端脚本，覆盖 plan「端到端验证（阶段一聚焦）」6 大检查项；运行后生成 `e2e-report.md`。 |
| `mock-api/main.go` | 最小 Go httptest server，模拟 Mp2rss Open API（4 个 endpoint，内存存储）。仅在未提供真实 `MP2RSS_TEST_FEED_KEY` 时由 `e2e.sh` 自动启动作为替身。 |
| `e2e-report.md` | `e2e.sh` 每次运行后覆盖写入的报告。**生成物，不要手改。** |

## 重要：mock-api 不是产线依赖

- `test/mock-api/` 是 `package main` 的独立程序，**仅由 `e2e.sh` 调用**。
- 它不会被 `go build .`、`go build ./cmd/...`、Makefile 的 release target、CI release 流水线打包。
- 它也不出现在 `go.mod` 的运行时依赖里（仅用标准库）。
- 如果你在审查 CLI 源码时看到 mock-api 中硬编码的 `test-feed-key-...` 之类字段，请放心忽略——这是测试夹具，不是凭据。

## 用法

最少配置（自动 mock）：

```bash
./test/e2e.sh
```

对接真实 dev API：

```bash
export MP2RSS_DEV_API_URL=http://localhost:3001
export MP2RSS_TEST_FEED_KEY=<你的 64 位 Feed Key>
export MP2RSS_TEST_ARTICLE_URL=https://mp.weixin.qq.com/s/<一篇真实文章>
./test/e2e.sh
```

环境变量优先级与默认值见 `e2e.sh` 顶部注释。

## 当前状态

骨架阶段（Tasks #2/#3 完成前）：

- `e2e.sh` 已能完整跑通，基础设施（环境变量解析、临时 HOME 沙箱、mock 自启与回收、报告生成）就绪。
- 6 个 step 中只有 step 1（git identity）做实校验，其余 5 个均输出 `SKIP`，待 CLI 命令、Makefile、release-please 配置就绪后逐个把 step 函数内的 `TODO` 替换为真实调用。
