# mp2rss-cli

[Mp2RSS](https://mp2rss.bugcode.dev) 的命令行客户端，订阅微信公众号、查看文章、管理订阅。

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

也可在 [Releases](https://github.com/areyoubugcoder/mp2rss-cli/releases/latest) 直接下载对应平台二进制。完整安装与卸载说明见 [文档站 · 安装](https://areyoubugcoder.github.io/Mp2RSS/cli/install)。

## 用法

```bash
mp2rss auth login
mp2rss mp subscribe https://mp.weixin.qq.com/s/xxxxxxxxxx
mp2rss mp list
mp2rss mp list -o json | jq '.items[].mpName'
```

## 命令一览

| 命令 | 说明 |
| ---- | ---- |
| `mp2rss auth login` | 登录（默认浏览器 + loopback；支持 `-k` / `--no-browser`） |
| `mp2rss auth logout` | 清空本地 Feed 密钥 |
| `mp2rss auth status` | 查看登录状态 |
| `mp2rss mp list` | 列出订阅（`-q` 模糊搜索） |
| `mp2rss mp search <keyword>` | `mp list -q` 的语法糖 |
| `mp2rss mp subscribe <article-url>` | 订阅公众号 |
| `mp2rss mp remove <mpId>` | 取消订阅（`-y` 跳过确认） |
| `mp2rss mp articles <mpId>` | 历史文章 |
| `mp2rss update` | 自更新（`--check` / `--force`） |

完整命令参考见 [文档站 · 命令参考](https://areyoubugcoder.github.io/Mp2RSS/cli/commands)。

## 配置

本地配置 `~/.mp2rss/config.json`（目录 `0700`、文件 `0600`）：

```json
{
  "feed_key": "9f3a2c...（64 位 hex）",
  "api_url": "https://mp2rss.bugcode.dev",
  "last_login_at": 1747194198,
  "last_verify_at": 1747194198
}
```

环境变量 `MP2RSS_FEED_KEY` / `MP2RSS_API_URL`，优先级高于配置文件、低于命令行 flag。

## 文档

- 简介：<https://areyoubugcoder.github.io/Mp2RSS/cli/>
- 安装：<https://areyoubugcoder.github.io/Mp2RSS/cli/install>
- 登录：<https://areyoubugcoder.github.io/Mp2RSS/cli/login>
- 命令参考：<https://areyoubugcoder.github.io/Mp2RSS/cli/commands>
- FAQ：<https://areyoubugcoder.github.io/Mp2RSS/cli/faq>
- API 参考：<https://mp2rss.bugcode.dev/>

## 在 Claude Code 中使用

mp2rss 提供 Claude Code agent skills，让你在 Claude Code 里用自然语言调用 CLI。在 Claude Code 中：

```
/plugin marketplace add areyoubugcoder/mp2rss-cli
/plugin install mp2rss-cli@mp2rss
```

之后可以直接说：

- 「登录公众号 RSS 服务」
- 「订阅这个公众号 https://mp.weixin.qq.com/s/...」
- 「我订阅了哪些公众号」
- 「这个公众号 <mpId> 最近发了什么」

包含两个 skill：[`mp2rss-auth`](skills/mp2rss-auth/SKILL.md)（登录态管理）与 [`mp2rss-mp`](skills/mp2rss-mp/SKILL.md)（订阅与文章）。

## License

[MIT](LICENSE)
