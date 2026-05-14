# Changelog

## 1.0.0 (2026-05-14)


### ✨ 新功能

* add one-shot install.sh for Linux / macOS ([69d7ce5](https://github.com/areyoubugcoder/mp2rss-cli/commit/69d7ce58bae0b167af0fc400c60f2b5f28340c93))
* **npm:** add @mp2rss/cli wrapper with platform-aware postinstall ([1d1a000](https://github.com/areyoubugcoder/mp2rss-cli/commit/1d1a000ff076617f3d77c7edf02b71c9f295ca11))
* scaffold cobra CLI with auth, mp, and update commands ([26ce776](https://github.com/areyoubugcoder/mp2rss-cli/commit/26ce77682018765c1edf5b60a3996805f9c40c53))
* **update:** real self-update with checksum verify and atomic replace ([ea68136](https://github.com/areyoubugcoder/mp2rss-cli/commit/ea68136811d9050ad97080922cff39c2252fbe7d))


### 🐛 修复

* **authflow:** handle CORS preflight and align dev origin to localhost:3000 ([8b6e722](https://github.com/areyoubugcoder/mp2rss-cli/commit/8b6e722691bac334727d9fd9aa0629d90e4f0ec7))


### ♻️ 重构

* **output:** unify JSON keys to camelCase and timestamps to unix ms ([c41880d](https://github.com/areyoubugcoder/mp2rss-cli/commit/c41880d70de8fb300bc48d1ec81a0c233b234b3b))


### 📚 文档

* add README with badges, install guide, command overview ([4b15c83](https://github.com/areyoubugcoder/mp2rss-cli/commit/4b15c837a5b30315d368333bfe24a500e179f8f3))
* **readme:** note -y short flag for mp remove ([43db2fb](https://github.com/areyoubugcoder/mp2rss-cli/commit/43db2fbe7f9bcee117937b33503a5ba78848d98c))
* **readme:** rewrite install section for stage-two delivery ([8e9c93e](https://github.com/areyoubugcoder/mp2rss-cli/commit/8e9c93ee9d3be3d3c6534a6c92a55db79c369a42))


### 🔧 构建

* add Makefile, golangci-lint config, CI + release-please workflows ([8a193e4](https://github.com/areyoubugcoder/mp2rss-cli/commit/8a193e48c2eb772931ce92cfa0545d43377b4a1b))


### 👷 持续集成

* add release workflow for multi-arch builds + npm publish ([a5ec824](https://github.com/areyoubugcoder/mp2rss-cli/commit/a5ec824d492d32bcbb4ca6604aacb783f009af4b))
* drop unused version var in release.yml build step ([f984178](https://github.com/areyoubugcoder/mp2rss-cli/commit/f9841787b7bd2a6f0e9f75d0d4b1a6a8bac1b296))


### ✅ 测试

* cover config / output / client / authflow critical paths ([94efbb1](https://github.com/areyoubugcoder/mp2rss-cli/commit/94efbb1f30d1d1cd972319bb5960209d37124a54))
* scaffold e2e harness and mock api ([02a1e8a](https://github.com/areyoubugcoder/mp2rss-cli/commit/02a1e8a29ccf4e84aa3634df6fca3de2aa87f00c))
* **stage2:** static validation for release workflow / install.sh / npm / update ([8c4fb94](https://github.com/areyoubugcoder/mp2rss-cli/commit/8c4fb94e7e763981c051b1f21749dc8f6c4f61bf))
* wire up e2e steps against real CLI ([12f05d6](https://github.com/areyoubugcoder/mp2rss-cli/commit/12f05d6d655679d808c93ea4d4f89a0121dbfebd))

<!--
本项目历史变更见上方版本条目。v1.0.0 由早期 release-please 工具自动生成；
后续版本改为 push `v*` tag 时由 GitHub 自动生成 release notes，不再写入此文件。
版本号遵循 Semantic Versioning。
-->
