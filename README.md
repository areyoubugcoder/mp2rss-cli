# mp2rss-cli

[Mp2RSS](https://mp2rss.bugcode.dev) 的命令行客户端 —— WeChat 公众号 → RSS feed，命令行管理订阅、查看历史文章。AI Agent 友好（Claude Code / Cursor / OpenClaw skills）。Keywords: wechat, weixin, 微信, 公众号, rss, feed, subscription, 订阅.

[![Release](https://img.shields.io/github/v/release/areyoubugcoder/mp2rss-cli?display_name=tag&sort=semver)](https://github.com/areyoubugcoder/mp2rss-cli/releases)
[![Downloads](https://img.shields.io/github/downloads/areyoubugcoder/mp2rss-cli/total)](https://github.com/areyoubugcoder/mp2rss-cli/releases)
[![npm](https://img.shields.io/npm/v/@mp2rss/cli)](https://www.npmjs.com/package/@mp2rss/cli)
[![npm downloads](https://img.shields.io/npm/dm/@mp2rss/cli)](https://www.npmjs.com/package/@mp2rss/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/areyoubugcoder/mp2rss-cli)](https://goreportcard.com/report/github.com/areyoubugcoder/mp2rss-cli)

## 安装

```bash
# 一键安装脚本（macOS / Linux）
curl -fsSL https://raw.githubusercontent.com/areyoubugcoder/mp2rss-cli/main/scripts/install.sh | sh

# npm（Node ≥ 18）
pnpm add -g @mp2rss/cli
```

也可在 [Releases](https://github.com/areyoubugcoder/mp2rss-cli/releases/latest) 直接下载对应平台二进制。完整安装/卸载说明见 [文档站 · 安装](https://areyoubugcoder.github.io/Mp2RSS/cli/install)。

升级到最新版本：

```bash
mp2rss update           # 检查并升级
mp2rss update --check   # 只检查不升级
```

## CLI 如何使用

### 1. 安装 CLI

参考上面 [安装](#安装) 章节。

### 2. 登录

```bash
mp2rss auth login
```

默认浏览器 loopback 授权，登录后凭证写入 `~/.mp2rss/config.json`。也可用 `mp2rss auth login -k <feed-key>` 直传 Feed Key（适合 CI / 无头环境），或 `--no-browser` 仅打印授权 URL。查看登录态：`mp2rss auth status`。

### 3. 使用

```bash
mp2rss mp subscribe https://mp.weixin.qq.com/s/xxxxxxxxxx   # 订阅一个公众号
mp2rss mp list                                              # 列出订阅
mp2rss mp list -q 财经                                      # 模糊搜索已订阅源
mp2rss mp articles <mpId>                                   # 看该公众号历史文章
mp2rss mp list -o json | jq '.items[].mpName'              # 结构化数据 + jq 处理
```

> ⚠️ 订阅时传入的是 **公众号文章的 URL**（`https://mp.weixin.qq.com/s/...`），不是公众号名字。

## 完整命令参考

### 认证

| 命令 | 说明 |
|------|------|
| `mp2rss auth login` | 浏览器 loopback 登录；`-k <feed-key>` 直传；`--no-browser` 仅打印授权 URL |
| `mp2rss auth status` | 查看登录状态、Feed Key 来源（env / config / none）、API URL、最近登录时间 |
| `mp2rss auth logout` | 清空本地 Feed Key（保留 API URL 配置） |

### 公众号

| 命令 | 说明 |
|------|------|
| `mp2rss mp list` | 列出订阅；`-q <keyword>` 模糊搜索，`-p <page>` 翻页，`--page-size <n>` 每页条数（最大 50） |
| `mp2rss mp search <keyword>` | `mp list -q` 的语法糖，flag 与输出一致 |
| `mp2rss mp subscribe <article-url>` | 订阅公众号；参数必须是 `mp.weixin.qq.com/s/...` 文章 URL |
| `mp2rss mp remove <mpId>` | 按 mpId 取消订阅；`-y` 跳过确认 |
| `mp2rss mp articles <mpId>` | 列出指定公众号的历史文章；`-p` / `--page-size`（最大 100） |

### 命令表格说明

- 所有命令默认 `-o table`，加 `-o json` 输出结构化数据（`auth login` 例外，仅文本反馈）
- JSON 错误形态统一为 `{"error":{"message":"...","code":<int>}}`，`code` 为 HTTP 状态码或 CLI exit code
- Exit codes：`0` 成功 / `1` 通用错误（网络）/ `2` 参数错误 / `3` 鉴权失败 / `4` 资源不存在 / `5` 上游不可用

## 全局参数

所有子命令都支持以下持久化 flag：

| Flag | 等价环境变量 | 说明 |
|------|-------------|------|
| `-o, --output <table\|json>` | — | 输出格式，默认 `table` |
| `--api-key <feed-key>` | `MP2RSS_FEED_KEY` | 覆盖 Feed Key |
| `--api-url <url>` | `MP2RSS_API_URL` | 覆盖 API 地址（默认 `https://mp2rss.bugcode.dev`） |

优先级（高 → 低）：命令行 flag > 环境变量 > 配置文件 > 默认值。

## 配置

本地配置 `~/.mp2rss/config.json`：

```json
{
  "feed_key": "9f3a2c...（64 位 hex）",
  "api_url": "https://mp2rss.bugcode.dev",
  "last_login_at": 1747194198,
  "last_verify_at": 1747194198,
  ...
}
```

Feed Key 可在 <https://mp2rss.bugcode.dev/> 登录后查看或重置。FAQ / 故障排查见 [文档站 · FAQ](https://areyoubugcoder.github.io/Mp2RSS/cli/faq)。

## AI Agent 如何使用

mp2rss 在 `skills/` 目录提供两个 agent skill —— [`mp2rss-auth`](skills/mp2rss-auth/SKILL.md)（登录态管理）和 [`mp2rss-mp`](skills/mp2rss-mp/SKILL.md)（订阅与文章），让 AI Agent 用自然语言驱动 mp2rss CLI。

### 安装

```bash
# 1. npx skills（Claude Code / Cursor 通用，最简）
npx skills add areyoubugcoder/mp2rss-cli -y -g
```

```
# 2. Claude Code 内置 plugin marketplace
/plugin marketplace add areyoubugcoder/mp2rss-cli
/plugin install mp2rss-cli@mp2rss
```

```bash
# 3. OpenClaw🦞（https://clawhub.ai/mp2rss/mp2rss-cli）
clawhub package install mp2rss-cli
```

### 使用举例

安装后在 AI 客户端里直接说自然语言，agent 会自动调用对应 skill：

- 「登录公众号 RSS 服务」/「我的 Feed Key 是什么」→ `mp2rss-auth`
- 「订阅这个公众号 https://mp.weixin.qq.com/s/...」→ `mp2rss mp subscribe`
- 「我订阅了哪些公众号」/「搜一下我订阅的财经类公众号」→ `mp2rss mp list / search`
- 「看一下 X 这个号的最新文章」→ `mp2rss mp articles`
- 「取消订阅公众号 X」/「把 X 从订阅里删了」→ `mp2rss mp remove`

所有命令支持 `-o json`，Agent 可直接解析结构化输出做后续处理。

## License

[MIT](LICENSE)
