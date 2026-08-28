# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目本质

这是一个 **Go 编写的 CLI**（`mp2rss`），把微信公众号与 X 账号转成 RSS/Atom/JSON Feed。

- 入口 `main.go` → `cmd.Execute()`，命令树在 `cmd/` 下（`auth` / `mp` / `x` / `update`），用 [cobra](https://github.com/spf13/cobra)。
- 业务逻辑都在 `internal/`（`authflow`、`client`、`config`、`output`、`errs` 等），不要在 `cmd/` 里堆逻辑。
- `npm/` 只是发布用的薄壳：`postinstall` 时下载对应平台的 Go 二进制，**不包含任何业务逻辑**。改功能改 Go，不要改 `npm/`。动到 `npm/` 时用 **pnpm**，不要用 npm/yarn。

## Agent Skills 双形态（skills/ 与 openclaw/）

同一套 agent 使用说明以**两种形态**存在于本仓库，面向不同生态，**改 CLI 行为时必须两边同步**：

- `skills/mp2rss-{auth,mp,x}/SKILL.md` —— Claude Code / Cursor 形态，按域拆成 3 个 skill。经 `.claude-plugin/`（plugin marketplace）与 `npx skills add areyoubugcoder/mp2rss-cli` 分发。
- `openclaw/mp2rss/` —— OpenClaw / ClawHub 形态：单入口 `SKILL.md`（路由）+ `references/{auth,mp,x,errors,install}.md`。由 `mp2rss-openclaw` 独立仓库迁入（该仓库已废弃），发布到 ClawHub，slug 为 `mp2rss`。**必须保持两层嵌套**（`openclaw/mp2rss/SKILL.md` 而非 `openclaw/SKILL.md`）：`npx skills add` 会把仓库根下一层深的 `*/SKILL.md` 当作 skill 安装，嵌套两层才不会被 Claude 侧安装误捡（已实测验证）。

同步纪律（延伸自「协议改动必须同步文档」）：任何命令 / flag / JSON 字段 / 退出码 / 错误 envelope 的改动，`skills/` 与 `openclaw/mp2rss/references/` 都要改，**以 CLI 实际行为为准**（不确定就跑 `./mp2rss ... -o json` 实测，不要照抄旧文档）。已知易错点：HTTP 403 / 429 均映射到 exit 1（`internal/client` 未特判），不是 3 / 5。

ClawHub 发布流程（skills 内容有实质变更时）：

1. bump `openclaw/mp2rss/package.json#version` 与 `openclaw/mp2rss/SKILL.md` frontmatter 的 `version`（两处保持一致）；
2. `clawhub skill publish openclaw/mp2rss --slug mp2rss --version <X.Y.Z> --changelog "<中文变更说明>"`（需 `clawhub login` 登录态）。

ClawHub 版本号独立于 CLI 版本（release-please 管 CLI，ClawHub 手动发），不要混用。

## 常用命令（Go 1.21+）

- 构建：`make build`（带 trimpath/ldflags，产物 `./mp2rss`）；交叉编译 `make build-all`（输出到 `dist/`）。
- 测试：`make test`（即 `go test ./...`）；跑单个测试用 `go test -run TestName ./internal/xxx`。
- 端到端：`./test/e2e.sh`，默认走 `test/mock-api/` 的 mock。需要打真实 API 时设 `MP2RSS_TEST_FEED_KEY` / `MP2RSS_DEV_API_URL` / `MP2RSS_TEST_ARTICLE_URL`。
- Lint：`make lint`（`go vet ./...` + `golangci-lint run ./...`，配置见 `.golangci.yml`，需本机装有 `golangci-lint`）。

## 代码风格

- Go 用 `gofmt`（Tab 缩进）。`.editorconfig` 里非 Go 文件是 2 空格、LF。
- 导出符号必须有文档注释、每个包要有包注释（revive 已开启 `exported` / `package-comments` / `var-naming`）。

## 易踩的坑

- **ID 精度**：JSON 里 `mpId` 是 int64（数字），`xUserId` 是字符串。解析时把 `xUserId` 当字符串处理，别让 JS/JSON 数字精度截断。
- **公众号订阅**：`mp subscribe` 接收的是**文章 URL**（`https://mp.weixin.qq.com/s/...`），不是账号名或二维码。
- **X 是只读**：CLI 只有 `x list` / `x posts` / `x articles`。X 的订阅/取消/搜索只能在 Web 控制台（`https://mp2rss.bugcode.dev`）完成，不要在 CLI 里加这些命令。
- **凭据**：登录态存在 `~/.mp2rss/config.json`（目录 0700 / 文件 0600），其中的 `feedKey` 是敏感凭据 —— 绝不能打日志、写进源码或提交到仓库。
- 配置优先级：命令行 flag（`--api-key` / `--api-url`）> 环境变量（`MP2RSS_FEED_KEY` / `MP2RSS_API_URL`）> 配置文件。

## 退出码约定（`internal/errs`）

`0` 成功 · `1` 通用错误 · `2` 参数错误 · `3` 未认证 · `4` 资源不存在 · `5` 上游 API 不可用。新增错误路径时复用这些码。

## 协作约定

- 提交信息用 Conventional Commits + 中文描述，例如 `feat(mp): 新增订阅命令`、`fix(authflow): 修正 CORS 白名单`。
- 允许直接提交到 `main`。
- 版本发布走 `release-please`（见 `.github/workflows/release.yml`），不要手动改版本号。
