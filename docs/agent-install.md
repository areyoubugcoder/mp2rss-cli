# mp2rss AI Agent 安装指南

本指南面向 AI Agent（Claude Code / Cursor / Cline 等支持 [skills](https://www.npmjs.com/package/skills) 与 SKILL.md 的客户端）。完成后，Agent 可用自然语言驱动 mp2rss CLI —— 订阅微信公众号、列出 / 搜索订阅源、查询公众号历史文章。

> 以下步骤面向 AI Agent，部分步骤需要用户在浏览器中配合完成（如登录授权）。

## 环境要求

- **Node.js ≥ 18**（提供 `npm` / `npx`）—— 用于安装 `mp2rss` CLI 和 agent skills
- **macOS / Linux / Windows** 任一即可（CLI 提供 6 个平台预编译二进制）
- 浏览器（用于第 2 步 loopback 授权登录；CI / 无头环境可改用 Feed Key 直传，见下方说明）

> 仅在从源码构建 mp2rss CLI 时才需要 Go ≥ 1.21；通过 npm / 安装脚本 / Release 下载二进制均无需 Go。

## 第 1 步 安装 mp2rss CLI

任选其一：

```bash
# 方式 A：npm（推荐，跨平台一致；Node ≥ 18）
pnpm add -g @mp2rss/cli

# 方式 B：一键脚本（macOS / Linux，自动选平台二进制）
curl -fsSL https://raw.githubusercontent.com/areyoubugcoder/mp2rss-cli/main/scripts/install.sh | sh
```

也可直接从 [Releases](https://github.com/areyoubugcoder/mp2rss-cli/releases/latest) 下载对应平台的二进制并加入 `PATH`。

验证 CLI 可用：

```bash
mp2rss --version
```

## 第 2 步 安装 Agent Skills

mp2rss 提供两个独立 skill：

| Skill | 作用 |
|-------|------|
| [`mp2rss-auth`](../skills/mp2rss-auth/SKILL.md) | 登录态管理：`auth login` / `auth status` / `auth logout` |
| [`mp2rss-mp`](../skills/mp2rss-mp/SKILL.md) | 公众号订阅与文章：`mp subscribe` / `mp list` / `mp search` / `mp remove` / `mp articles` |

任选其一安装到你的 AI 客户端：

```bash
# 方式 A：npx skills（Claude Code / Cursor 通用，最简，一次装齐两个 skill）
npx skills add areyoubugcoder/mp2rss-cli -y -g

# 方式 B：Claude Code 内置 plugin marketplace
/plugin marketplace add areyoubugcoder/mp2rss-cli
/plugin install mp2rss-cli@mp2rss

# 方式 C：OpenClaw 🦞 单 skill 安装（按需）
openclaw skills install mp2rss-auth
openclaw skills install mp2rss-mp
```

> Skill 是 SKILL.md 形态的轻量描述文件，不携带任何二进制；Agent 在识别到匹配语义时按 SKILL.md 中描述调用 `mp2rss` 命令。因此**第 1 步的 `mp2rss` 二进制必须先安装**，否则 skill 调用会失败。

## 第 3 步 登录

```bash
mp2rss auth login
```

默认走浏览器 loopback 授权：CLI 启动本地一次性 HTTP 服务，打开浏览器登录 [mp2rss.bugcode.dev](https://mp2rss.bugcode.dev/)，回调写入 `~/.mp2rss/config.json`（目录 `0700` / 文件 `0600`）。

无头 / CI / 远程场景：

```bash
# 直传 Feed Key（在 https://mp2rss.bugcode.dev/ 登录后查看或重置）
mp2rss auth login -k <your-feed-key>

# 远程模式：不打开浏览器，CLI 仅打印授权 URL，复制到本地浏览器完成后回填
mp2rss auth login --no-browser
```

> Agent 拿到 `auth login --no-browser` 的输出后，应**完整提取授权 URL 转发给用户**，并提示「请在浏览器打开此链接，登录后把页面上的 Feed Key 粘回」。

## 第 4 步 验证

```bash
mp2rss auth status -o json
```

预期输出（`loggedIn: true` 即配置完成）：

```json
{
  "loggedIn": true,
  "source": "config",
  "apiUrl": "https://mp2rss.bugcode.dev",
  "feedKeyMasked": "abcdef***",
  "lastLoginAt": 1747194198000,
  "lastVerifyAt": 1747194199000
}
```

`source` 取值：`"env"`（环境变量 `MP2RSS_FEED_KEY`）/ `"config"`（配置文件）/ `"none"`（未配置）。Agent 调用 `mp2rss mp` 任何子命令前如不确定状态，应先跑 `mp2rss auth status -o json` 解析 `loggedIn` 字段。

## 第 5 步 在 Agent 里使用

安装完成后，直接对 AI 客户端说自然语言：

| 用户说 | Agent 自动调用 |
|--------|---------------|
| 「登录公众号 RSS 服务」/「我的 Feed Key 是什么」 | `mp2rss auth login` / `mp2rss auth status` |
| 「订阅这个公众号 `https://mp.weixin.qq.com/s/...`」 | `mp2rss mp subscribe <article-url>` |
| 「我订阅了哪些公众号」/「搜一下我订阅的财经类公众号」 | `mp2rss mp list` / `mp2rss mp search 财经` |
| 「看下 X 这个号的最新文章」 | `mp2rss mp articles <mpId>` |
| 「取消订阅公众号 X」/「把 X 从订阅里删了」 | `mp2rss mp remove <mpId>` |

⚠️ **订阅参数最常犯的错**：用户说「订阅这个公众号 X」时，`X` 必须是 `mp.weixin.qq.com/s/...` **文章 URL**，不是公众号名字、二维码、或公众号主页链接。Agent 识别不到合法文章 URL 时应**先反问用户索要任意一篇文章链接**。

所有命令支持 `-o json`，Agent 可直接解析结构化输出做后续处理。错误统一为 `{"error":{"message":"...","code":<int>}}`；Exit codes：`0` 成功 / `1` 网络通用错误 / `2` 参数错误 / `3` 鉴权失败 / `4` 资源不存在 / `5` 上游不可用。

## 延伸

- CLI 完整命令参考：[README](../README.md)
- Skill 详细字段与 JSON shape：[`skills/mp2rss-auth/SKILL.md`](../skills/mp2rss-auth/SKILL.md) / [`skills/mp2rss-mp/SKILL.md`](../skills/mp2rss-mp/SKILL.md)
- FAQ / 故障排查：[文档站 · FAQ](https://areyoubugcoder.github.io/Mp2RSS/cli/faq)
