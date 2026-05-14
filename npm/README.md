# @mp2rss/cli

[mp2rss](https://mp2rss.com) 官方命令行工具——在终端里完成订阅、查询、管理微信公众号的全部操作。

本 npm 包是 [mp2rss-cli](https://github.com/areyoubugcoder/mp2rss-cli) Go 二进制的包装，安装时会按平台从 GitHub Releases 下载对应可执行文件并放在 `node_modules/@mp2rss/cli/bin/`。

## 安装

```bash
# 全局
npm install -g @mp2rss/cli

# 或 pnpm
pnpm add -g @mp2rss/cli
```

支持的平台：

| OS      | amd64 | arm64 |
|---------|:-----:|:-----:|
| macOS   | ✅    | ✅    |
| Linux   | ✅    | ✅    |
| Windows | ✅    | -     |

## 快速上手

```bash
mp2rss auth login
mp2rss mp subscribe https://mp.weixin.qq.com/s/xxxxxxxxxx
mp2rss mp list
mp2rss mp list -o json | jq '.items[].mpName'
```

## 环境变量

| 变量 | 作用 |
| ---- | ---- |
| `MP2RSS_VERSION` | 安装时强制使用指定版本（默认读取 package.json `version`） |
| `MP2RSS_NO_VERIFY` | 安装时跳过 SHA-256 校验（不推荐） |
| `MP2RSS_FEED_KEY` | 运行时覆盖 Feed Key |
| `MP2RSS_API_URL` | 运行时覆盖 API 地址 |

## 卸载

```bash
npm uninstall -g @mp2rss/cli
```

CLI 自身的本地配置 `~/.mp2rss/config.json` 需要手动删除（避免误清账户绑定）。

## License

MIT
