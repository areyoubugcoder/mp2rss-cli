# Mp2rss-cli（微信公众号订阅，公众号转RSS、JSON）

[Mp2RSS](https://mp2rss.bugcode.dev) 的命令行客户端 —— 管理订阅、订阅微信公众号，文章转RSS、JSON，查看历史文章。AI Agent 友好（Claude Code / Cursor skills）。Keywords: wechat, weixin, 微信, 公众号, rss, feed, subscription, 订阅.

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

## 使用

```bash
mp2rss auth login                                            # 浏览器登录
mp2rss mp subscribe https://mp.weixin.qq.com/s/xxxxxxxxxx    # 订阅公众号（参数是文章 URL）
mp2rss mp list                                               # 列出已订阅公众号
mp2rss mp articles <mpId>                                    # 看历史文章
mp2rss x list                                                # 列出已订阅 X 账号
mp2rss x posts <xUserId>                                     # 看推文流
```

📖 完整命令、flag、JSON 输出、错误码、配置文件、全局参数等详细说明 → **[在线文档 · CLI](https://areyoubugcoder.github.io/Mp2RSS/cli/)**

常用快捷入口：
[安装](https://areyoubugcoder.github.io/Mp2RSS/cli/install)
· [auth 命令](https://areyoubugcoder.github.io/Mp2RSS/cli/auth)
· [mp 命令](https://areyoubugcoder.github.io/Mp2RSS/cli/mp)
· [x 命令](https://areyoubugcoder.github.io/Mp2RSS/cli/x)
· [FAQ](https://areyoubugcoder.github.io/Mp2RSS/cli/faq)

## AI Agent 如何使用

Mp2rss 在 `skills/` 目录提供两个 agent skill —— [`mp2rss-auth`](skills/mp2rss-auth/SKILL.md)（登录态管理）和 [`mp2rss-mp`](skills/mp2rss-mp/SKILL.md)（订阅与文章），让 AI Agent 用自然语言驱动 Mp2rss CLI。完整安装与使用步骤见 **[AI Agent 安装指南](docs/agent-install.md)**。

### 快速安装

```bash
# 1. 装 CLI（Node ≥ 18）
pnpm add -g @mp2rss/cli

# 2. 装 Skills，任选其一
npx skills add areyoubugcoder/mp2rss-cli -y -g          # Claude Code / Cursor 通用，一次装齐两个 skill
/plugin marketplace add areyoubugcoder/mp2rss-cli       # Claude Code 内置 plugin marketplace
/plugin install mp2rss-cli@mp2rss

# 3. 登录
mp2rss auth login
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
