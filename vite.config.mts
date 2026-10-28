import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

import { comicsBaseRedirectPlugin } from './dev/proxy/comicsBaseRedirect';
import { goApiDevProxy } from './dev/proxy/viteProxy';

const rootDir = path.dirname(fileURLToPath(import.meta.url));

/** Comma-separated extra hostnames (VITE_ALLOWED_HOSTS=foo.bar.ts.net) */
function extraAllowedHosts(): string[] {
  return (process.env.VITE_ALLOWED_HOSTS ?? '')
    .split(',')
    .map((h) => h.trim())
    .filter(Boolean);
}

/** Tailscale MagicDNS and local dev hostnames allowed by Vite's host check. */
const devAllowedHosts = ['.ts.net', '.localhost', ...extraAllowedHosts()];

function getHttpsConfig() {
  if (process.env.HTTPS !== 'true') return undefined;

  const keyPath = process.env.SSL_KEY_FILE || './tls/comics.key';
  const certPath = process.env.SSL_CRT_FILE || './tls/comics.crt';
  const key = path.resolve(rootDir, keyPath);
  const cert = path.resolve(rootDir, certPath);

  if (!fs.existsSync(key) || !fs.existsSync(cert)) return undefined;

  return {
    key: fs.readFileSync(key),
    cert: fs.readFileSync(cert),
  };
}

export default defineConfig({
  base: '/comics/',
  plugins: [react(), comicsBaseRedirectPlugin()],
  resolve: {
    alias: {
      '@pb': path.resolve(rootDir, 'src/frontend/pb'),
      '@config': path.resolve(rootDir, 'config'),
    },
  },
  build: {
    outDir: 'build',
  },
  server: {
    https: getHttpsConfig(),
    proxy: goApiDevProxy(rootDir),
    allowedHosts: devAllowedHosts,
  },
  preview: {
    proxy: goApiDevProxy(rootDir),
    allowedHosts: devAllowedHosts,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/setupTests.ts',
  },
});
