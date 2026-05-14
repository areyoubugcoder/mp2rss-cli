# mp2rss-cli

`mp2rss` 是 [Mp2RSS](https://mp2rss.com) 官方命令行工具——在终端里完成订阅、查询、管理微信公众号的全部操作，无需打开浏览器控制台。

[![Release](https://img.shields.io/github/v/release/areyoubugcoder/mp2rss-cli?display_name=tag&sort=semver)](https://github.com/areyoubugcoder/mp2rss-cli/releases)
[![Downloads](https://img.shields.io/github/downloads/areyoubugcoder/mp2rss-cli/total)](https://github.com/areyoubugcoder/mp2rss-cli/releases)
[![npm](https://img.shields.io/npm/v/@mp2rss/cli)](https://www.npmjs.com/package/@mp2rss/cli)
[![npm downloads](https://img.shields.io/npm/dm/@mp2rss/cli)](https://www.npmjs.com/package/@mp2rss/cli)
[![CI](https://github.com/areyoubugcoder/mp2rss-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/areyoubugcoder/mp2rss-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/areyoubugcoder/mp2rss-cli)](https://goreportcard.com/report/github.com/areyoubugcoder/mp2rss-cli)

## 特性亮点

- **单文件零依赖**：Go 编译产物，体积约 10 MB，跨平台开箱即用。
- **浏览器授权登录**：默认走 OAuth Loopback 流（类似 `gh auth login` / `gcloud`），无需手动复制 Feed 密钥。
- **CI / 无头友好**：支持 `--feed-key` 直接落盘与 `--no-browser` 远程登录。
- **双输出模式**：所有命令同时支持 `table`（CJK 宽度感知）与 `json`（jq 友好），错误也走 stdout。
- **完整覆盖订阅 API**：列表 / 搜索 / 订阅 / 取消 / 历史文章一站到位。
- **安全默认**：loopback 仅监听 `127.0.0.1`、state 防 CSRF、Feed 密钥永不明文回显，配置文件 `0600`。

## 安装

### 本地源码构建（当前阶段唯一方式）

需要 Go 1.22+。

```bash
git clone https://github.com/areyoubugcoder/mp2rss-cli.git
cd mp2rss-cli
go build -o mp2rss .
./mp2rss --version
```

把 `mp2rss` 放到 `PATH` 中任意目录即可全局调用：

```bash
sudo mv mp2rss /usr/local/bin/
```

### v1.x 起将支持

首个正式版本（v1.x）发布后将开放以下安装方式：

```bash
# Homebrew（占位，上线后可用）
brew install areyoubugcoder/tap/mp2rss

# npm 包装（占位，上线后可用）
npm install -g @mp2rss/cli

# 一键安装脚本（占位，上线后可用）
curl -fsSL https://mp2rss.com/install.sh | sh
```

也可在 [Releases](https://github.com/areyoubugcoder/mp2rss-cli/releases) 页面按平台 / 架构直接下载二进制并校验 `checksums.txt`。

## 快速上手

```bash
# 1. 登录（浏览器会自动打开 Mp2RSS 授权页）
mp2rss auth login

# 2. 用任意一篇该公众号的文章链接发起订阅
mp2rss mp subscribe https://mp.weixin.qq.com/s/xxxxxxxxxx

# 3. 查看当前账户下的订阅
mp2rss mp list
mp2rss mp list -o json | jq '.items[].mpName'
```

更多登录方式（`-k/--feed-key` 直接落盘、`--no-browser` 远程登录）见 [文档站 · 登录](https://mp2rss.com/cli/login)。

## 命令一览

| 命令 | 说明 |
| ---- | ---- |
| `mp2rss auth login` | 登录（默认浏览器 + loopback；支持 `-k` / `--no-browser`） |
| `mp2rss auth logout` | 清空本地 Feed 密钥 |
| `mp2rss auth status` | 查看登录状态与 API 地址 |
| `mp2rss mp list` | 列出订阅（支持 `-q` 模糊搜索、分页） |
| `mp2rss mp search <keyword>` | `mp list -q` 的语法糖 |
| `mp2rss mp subscribe <article-url>` | 通过文章链接订阅公众号 |
| `mp2rss mp remove <mpId>` | 取消订阅（`-y/--yes` 跳过确认） |
| `mp2rss mp articles <mpId>` | 查询某公众号的历史文章 |
| `mp2rss update` | 自更新（v1.x 启用，当前为占位实现） |

完整参数、退出码、表格 / JSON 输出示例见 [文档站 · 命令参考](https://mp2rss.com/cli/commands)。

## 配置与环境变量

`mp2rss` 的本地配置位于 `~/.mp2rss/config.json`，写入时自动设置：

- 目录权限 `0700`（仅当前用户可访问）
- 文件权限 `0600`（仅当前用户可读写）

配置字段：

```json
{
  "feed_key": "9f3a2c...（64 位 hex）",
  "api_url": "https://api.mp2rss.com",
  "last_call_at": "2026-05-14T03:23:18Z"
}
```

环境变量：

| 变量 | 作用 | 优先级 |
| ---- | ---- | ------ |
| `MP2RSS_FEED_KEY` | 覆盖 Feed 密钥 | 高于配置文件、低于 `--api-key` |
| `MP2RSS_API_URL` | 覆盖 API 地址 | 高于配置文件、低于 `--api-url` |

## 文档

- 简介与场景：<https://mp2rss.com/cli/>
- 安装：<https://mp2rss.com/cli/install>
- 登录（三条路径与安全要点）：<https://mp2rss.com/cli/login>
- 命令参考：<https://mp2rss.com/cli/commands>
- FAQ：<https://mp2rss.com/cli/faq>
- API 参考：<https://mp2rss.com/api/>

## Contributing

欢迎 issue 与 PR。提交规范遵循 [Conventional Commits](https://www.conventionalcommits.org/zh-hans/v1.0.0/)：

- `feat:` 新功能
- `fix:` 修复
- `docs:` 文档
- `refactor:` 重构
- `build:` 构建 / 依赖
- `chore:` 其它杂项

仓库由 [release-please](https://github.com/googleapis/release-please) 自动聚合 commit 生成 CHANGELOG 与版本 PR，请确保 commit 消息符合规范。

详细开发流程见仓库 `.github/pull_request_template.md` 与 `CHANGELOG.md` 的维护说明。

## License

[MIT](LICENSE)
