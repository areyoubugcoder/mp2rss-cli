#!/usr/bin/env node
// mp2rss launcher.
//
// The real binary is downloaded by postinstall.cjs into ./bin/<exe>. This
// shim spawns it with the user's args, forwarding stdio and the exit code.
//
// Why a JS shim instead of a native bin? npm cross-platform installs a JS
// file as the entry, and Windows .cmd shims rely on a node executable — so
// `bin/mp2rss.js` is the lowest-friction form on all three OSes.
'use strict';

const { spawnSync } = require('node:child_process');
const path = require('node:path');
const fs = require('node:fs');

const binaryName = process.platform === 'win32' ? 'mp2rss.exe' : 'mp2rss';
const binaryPath = path.join(__dirname, binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error('✗ mp2rss 二进制未找到：' + binaryPath);
  console.error('  请重新安装：npm install -g @mp2rss/cli');
  console.error('  或参考：https://github.com/areyoubugcoder/mp2rss-cli#install');
  process.exit(1);
}

const res = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: false,
});

if (res.error) {
  console.error('✗ 启动 mp2rss 失败：' + res.error.message);
  process.exit(1);
}
process.exit(res.status === null ? 1 : res.status);
