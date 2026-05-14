#!/usr/bin/env node
// postinstall: download the mp2rss binary matching this host's OS/arch from
// the GitHub Release that corresponds to package.json:version, verify its
// SHA-256 against checksums.txt, and place it at ./bin/<exe>.
'use strict';

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const crypto = require('node:crypto');
const https = require('node:https');
const { spawnSync } = require('node:child_process');

const pkg = require('./package.json');
const REPO = 'areyoubugcoder/mp2rss-cli';
const VERSION = process.env.MP2RSS_VERSION || pkg.version;
const NO_VERIFY = !!process.env.MP2RSS_NO_VERIFY;

// Skip during local development checkouts (no real release published yet).
if (VERSION === '0.0.0') {
  console.log('[mp2rss] skipping binary download for placeholder version 0.0.0');
  process.exit(0);
}

const platformMap = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
const archMap = { x64: 'amd64', arm64: 'arm64' };
const goos = platformMap[process.platform];
const goarch = archMap[process.arch];
if (!goos || !goarch) {
  console.error(`[mp2rss] 暂不支持的平台：${process.platform}/${process.arch}`);
  process.exit(1);
}

const ext = goos === 'windows' ? '.zip' : '.tar.gz';
const binaryName = goos === 'windows' ? 'mp2rss.exe' : 'mp2rss';
const asset = `mp2rss-cli_${VERSION}_${goos}_${goarch}${ext}`;
const base = `https://github.com/${REPO}/releases/download/v${VERSION}`;
const archiveURL = `${base}/${asset}`;
const checksumsURL = `${base}/checksums.txt`;

const binDir = path.join(__dirname, 'bin');
const binaryPath = path.join(binDir, binaryName);

(async function main() {
  // Already installed at the right version? skip.
  if (fs.existsSync(binaryPath)) {
    const out = spawnSync(binaryPath, ['--version'], { encoding: 'utf8' });
    if (out.status === 0 && String(out.stdout).includes(VERSION)) {
      console.log(`[mp2rss] v${VERSION} 已存在，跳过下载`);
      return;
    }
  }

  fs.mkdirSync(binDir, { recursive: true });
  const tmpArchive = path.join(os.tmpdir(), `mp2rss-${process.pid}-${asset}`);

  console.log(`[mp2rss] 下载 ${asset}`);
  try {
    await downloadFile(archiveURL, tmpArchive);

    if (!NO_VERIFY) {
      console.log('[mp2rss] 校验 SHA-256');
      const sumsPath = path.join(os.tmpdir(), `mp2rss-${process.pid}-checksums.txt`);
      try {
        await downloadFile(checksumsURL, sumsPath);
        const expected = readChecksum(sumsPath, asset);
        if (!expected) throw new Error(`checksums.txt 中找不到 ${asset}`);
        const actual = sha256File(tmpArchive);
        if (actual.toLowerCase() !== expected.toLowerCase()) {
          throw new Error(`checksum 不匹配：expected ${expected}, got ${actual}`);
        }
      } finally {
        safeUnlink(sumsPath);
      }
    } else {
      console.log('[mp2rss] MP2RSS_NO_VERIFY=1 跳过校验');
    }

    console.log('[mp2rss] 解压');
    extract(tmpArchive, binDir, binaryName);
    fs.chmodSync(binaryPath, 0o755);
    console.log(`[mp2rss] ✓ 已安装：${binaryPath}`);
  } catch (err) {
    console.error('[mp2rss] ✗ 安装失败：' + err.message);
    console.error('         请重试 npm install -g @mp2rss/cli，或参考');
    console.error('         https://github.com/' + REPO + '#install');
    process.exit(1);
  } finally {
    safeUnlink(tmpArchive);
  }
})();

// ---------- helpers ----------

function downloadFile(url, dst, redirectsLeft = 5) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { 'user-agent': 'mp2rss-cli-npm' } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirectsLeft <= 0) return reject(new Error('too many redirects'));
          return downloadFile(res.headers.location, dst, redirectsLeft - 1).then(resolve, reject);
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const out = fs.createWriteStream(dst);
        res.pipe(out);
        out.on('finish', () => out.close(resolve));
        out.on('error', reject);
      })
      .on('error', reject);
  });
}

function readChecksum(path, asset) {
  const data = fs.readFileSync(path, 'utf8');
  for (const raw of data.split('\n')) {
    const line = raw.trim();
    if (!line || line.startsWith('#')) continue;
    const m = line.match(/^(\S+)\s+\*?(\S+)$/);
    if (!m) continue;
    const name = m[2];
    if (name === asset || name.endsWith('/' + asset)) return m[1];
  }
  return null;
}

function sha256File(p) {
  const buf = fs.readFileSync(p);
  return crypto.createHash('sha256').update(buf).digest('hex');
}

function extract(archive, dir, binaryName) {
  if (archive.endsWith('.zip')) {
    // Windows: use built-in tar (tar.exe supports zip since Win10) or PowerShell Expand-Archive.
    const r = spawnSync('tar', ['-xf', archive, '-C', dir, binaryName], { stdio: 'inherit' });
    if (r.status !== 0) {
      const ps = spawnSync(
        'powershell',
        ['-NoProfile', '-Command', `Expand-Archive -Path '${archive}' -DestinationPath '${dir}' -Force`],
        { stdio: 'inherit' },
      );
      if (ps.status !== 0) throw new Error('解压 ZIP 失败');
    }
  } else {
    const r = spawnSync('tar', ['-xzf', archive, '-C', dir, binaryName], { stdio: 'inherit' });
    if (r.status !== 0) throw new Error('解压 tar.gz 失败');
  }
  if (!fs.existsSync(path.join(dir, binaryName))) {
    throw new Error(`解压后未找到 ${binaryName}`);
  }
}

function safeUnlink(p) {
  try {
    fs.unlinkSync(p);
  } catch (_) {
    /* ignore */
  }
}
