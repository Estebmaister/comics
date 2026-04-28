import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

const rootDir = path.dirname(fileURLToPath(import.meta.url));

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
  plugins: [react()],
  resolve: {
    alias: {
      '@pb': path.resolve(rootDir, 'src/frontend/pb'),
    },
  },
  build: {
    outDir: 'build',
  },
  server: {
    https: getHttpsConfig(),
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/setupTests.ts',
  },
});

