# Mp2rss-cli（微信公众号 + X / Twitter 订阅管理，统一转 RSS / JSON）

> 🚀 **一行命令，把你关注的微信公众号和 X 账号变成 RSS，喂给你最熟悉的阅读器。**

[Mp2RSS](https://mp2rss.bugcode.dev) 的官方命令行客户端 —— 在终端订阅 / 查询 微信公众号与 X 账号、列出已订阅源、拉历史文章 / 推文流 / 长文流，输出 table 或 JSON。AI Agent 友好（Claude Code / Cursor skills），可作为 Web 控制台的脚本化替代。

> **关于 Mp2RSS** —— 把信任的信息源搬进熟悉的阅读器。公众号文章、X 推文 / 长文 → 标准 RSS 2.0 / Atom 1.0 / JSON Feed 1.1 / OPML 2.0，可同步到 Reeder、NetNewsWire、FreshRSS、Miniflux 等阅读器。

Keywords: wechat, weixin, 微信, 公众号, twitter, x, rss, atom, jsonfeed, opml, feed, subscription, 订阅, mp2rss.

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

# npm（Node ≥ 18，跨平台含 Windows）
pnpm add -g @mp2rss/cli
```

也可在 [Releases](https://github.com/areyoubugcoder/mp2rss-cli/releases/latest) 直接下载对应平台二进制，或 `go install` 从源码构建。完整安装/卸载说明见 [文档站 · 安装](https://areyoubugcoder.github.io/Mp2RSS/cli/install)。

升级到最新版本：

```bash
mp2rss update           # 检查并升级
mp2rss update --check   # 只检查不升级
```

## 使用

### 登录

```bash
mp2rss auth login        # 浏览器 loopback 授权（推荐）
mp2rss auth login -k <feed-key>   # 直传 Feed Key（CI / 无头）
mp2rss auth login --no-browser    # SSH 远程：仅打印授权 URL
```

### 微信公众号

```bash
mp2rss mp subscribe https://mp.weixin.qq.com/s/xxxxxxxxxx   # 参数是任意一篇文章的 URL
mp2rss mp list                                              # 已订阅公众号
mp2rss mp list -q 财经                                      # 模糊搜索已订阅源
mp2rss mp articles <mpId>                                   # 该公众号历史文章
mp2rss mp remove <mpId>                                     # 取消订阅
```

> ⚠️ 订阅公众号时传入的是 **文章 URL**（`https://mp.weixin.qq.com/s/...`），不是公众号名字。

### X（Twitter）

```bash
mp2rss x list                       # 已订阅 X 账号
mp2rss x posts <xUserId>            # 推文流
mp2rss x articles <xUserId>         # 长文流
```

> X 的搜索 / 订阅 / 取消订阅请在 [Web 控制台](https://mp2rss.bugcode.dev/) 完成，CLI 仅暴露读类操作。

### 脚本化

```bash
mp2rss mp list -o json | jq '.items[].mpName'   # 所有命令支持 -o json
```

📖 **完整命令参考**（全部 flag、JSON schema、错误码、配置文件、全局参数）→ **[在线文档 · CLI](https://areyoubugcoder.github.io/Mp2RSS/cli/)**

快捷入口：
[安装](https://areyoubugcoder.github.io/Mp2RSS/cli/install)
· [登录](https://areyoubugcoder.github.io/Mp2RSS/cli/login)
· [命令总览](https://areyoubugcoder.github.io/Mp2RSS/cli/commands)
· [FAQ](https://areyoubugcoder.github.io/Mp2RSS/cli/faq)

## AI Agent 如何使用

Mp2rss 以两种形态提供 agent skills，同仓库维护、内容同步：

**Claude Code / Cursor（`skills/` 目录，按域拆分）**

- [`mp2rss-auth`](skills/mp2rss-auth/SKILL.md) —— 登录态管理（login / logout / status）
- [`mp2rss-mp`](skills/mp2rss-mp/SKILL.md) —— 公众号订阅与文章查询
- [`mp2rss-x`](skills/mp2rss-x/SKILL.md) —— X 账号已订阅列表、推文流与长文流（只读 3 件套；订阅 / 取消订阅 / 搜索请到 Web 控制台）

**OpenClaw（[`openclaw/mp2rss/`](openclaw/mp2rss/) 目录，单入口路由 + references 子文档）**

- 发布在 ClawHub，slug `mp2rss`：`openclaw skills install mp2rss`（或 `clawhub install mp2rss`）

完整安装与使用步骤见 **[AI Agent 安装指南](docs/agent-install.md)**。

### 快速安装

```bash
# 1. 装 CLI（Node ≥ 18）
pnpm add -g @mp2rss/cli

# 2. 装 Skills，任选其一
mp2rss skills sync --global                             # CLI ≥ 1.2.0 内置，之后 mp2rss update 会顺带同步
npx skills add areyoubugcoder/mp2rss-cli -y -g          # Claude Code / Cursor 通用，一次装齐三个 skill
/plugin marketplace add areyoubugcoder/mp2rss-cli       # Claude Code 内置 plugin marketplace
/plugin install mp2rss-cli@mp2rss

# 3. 登录
mp2rss auth login
```

### 使用举例（自然语言驱动）

- 「登录公众号 RSS 服务」/「我的 Feed Key 是什么」→ `mp2rss-auth`
- 「订阅这个公众号 https://mp.weixin.qq.com/s/...」→ `mp2rss mp subscribe`
- 「我订阅了哪些公众号」/「搜一下我订阅的财经类公众号」→ `mp2rss mp list / search`
- 「<公众号名> 最近发了什么」→ `mp2rss mp articles`
- 「取消订阅 <公众号名>」→ `mp2rss mp remove`

所有命令支持 `-o json`，Agent 可直接解析结构化输出做后续处理。

## License

[MIT](LICENSE)
