# mp2rss-cli

[Mp2RSS](https://mp2rss.bugcode.dev) 的命令行客户端，订阅微信公众号、查看文章、管理订阅。

[![Release](https://img.shields.io/github/v/release/areyoubugcoder/mp2rss-cli?display_name=tag&sort=semver)](https://github.com/areyoubugcoder/mp2rss-cli/releases)
[![Downloads](https://img.shields.io/github/downloads/areyoubugcoder/mp2rss-cli/total)](https://github.com/areyoubugcoder/mp2rss-cli/releases)
[![npm](https://img.shields.io/npm/v/@mp2rss/cli)](https://www.npmjs.com/package/@mp2rss/cli)
[![npm downloads](https://img.shields.io/npm/dm/@mp2rss/cli)](https://www.npmjs.com/package/@mp2rss/cli)
[![CI](https://github.com/areyoubugcoder/mp2rss-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/areyoubugcoder/mp2rss-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/areyoubugcoder/mp2rss-cli)](https://goreportcard.com/report/github.com/areyoubugcoder/mp2rss-cli)

## 安装

```bash
# 一键安装脚本（macOS / Linux）
curl -fsSL https://mp2rss.bugcode.dev/install.sh | sh

# npm（Node ≥ 18）
pnpm add -g @mp2rss/cli

# 源码（Go ≥ 1.21）
git clone https://github.com/areyoubugcoder/mp2rss-cli.git && cd mp2rss-cli && make build
```

也可在 [Releases](https://github.com/areyoubugcoder/mp2rss-cli/releases/latest) 直接下载对应平台二进制。完整安装与卸载说明见 [文档站 · 安装](https://mp2rss.bugcode.dev/cli/install)。

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

完整命令参考见 [文档站 · 命令参考](https://mp2rss.bugcode.dev/cli/commands)。

## 配置

本地配置 `~/.mp2rss/config.json`（目录 `0700`、文件 `0600`）：

```json
{
  "feed_key": "9f3a2c...（64 位 hex）",
  "api_url": "https://mp2rss.bugcode.dev/api",
  "last_login_at": 1747194198,
  "last_verify_at": 1747194198
}
```

环境变量 `MP2RSS_FEED_KEY` / `MP2RSS_API_URL`，优先级高于配置文件、低于命令行 flag。

## 文档

- 简介：<https://mp2rss.bugcode.dev/cli/>
- 安装：<https://mp2rss.bugcode.dev/cli/install>
- 登录：<https://mp2rss.bugcode.dev/cli/login>
- 命令参考：<https://mp2rss.bugcode.dev/cli/commands>
- FAQ：<https://mp2rss.bugcode.dev/cli/faq>
- API 参考：<https://mp2rss.bugcode.dev/api/>

## License

[MIT](LICENSE)
