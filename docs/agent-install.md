# Mp2rss CLI 安装指南

以下步骤面向 AI Agent，部分步骤需要用户在浏览器中配合完成。

## 环境要求

开始安装之前，请确保环境中已安装：

- Node.js ≥ 18（npm/npx）
- Go ≥ 1.21（仅源码构建需要）

## 第 1 步 安装

```shell
# 安装 CLI
pnpm add -g @mp2rss/cli

# 安装 CLI SKILL（必需）
npx -y skills add areyoubugcoder/mp2rss-cli -y -g
```

CLI ≥ 1.2.0 也可用内置命令代替上面第二步：`mp2rss skills sync --global`。之后每次 `mp2rss update` 升级二进制时会顺带把本地 skills 同步到新版本（`--skip-skills` 跳过），`mp2rss skills status` 可随时查看是否漂移。

> OpenClaw 用户改用 ClawHub 安装 skill：`openclaw skills install mp2rss`（或 `clawhub install mp2rss`），对应本仓库 `openclaw/mp2rss/` 目录的单入口形态。

## 第 2 步 登录

Agent 运行以下命令，并提取授权链接发给用户。

```shell
mp2rss auth login
```

## 第 3 步 验证

```shell
mp2rss auth status
```

更多命令和能力指南，可参考 [Mp2rss CLI 命令参考](https://areyoubugcoder.github.io/Mp2RSS/cli/commands.html)。
