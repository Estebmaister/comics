import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import react from '@vitejs/plugin-react';
import type { Connect } from 'vite';
import { defineConfig, type Plugin } from 'vite';

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const goApiTarget = process.env.VITE_GO_API_TARGET ?? 'https://localhost:8081';

function comicsBaseRedirectPlugin(): Plugin {
  const redirect: Connect.NextHandleFunction = (req, res, next) => {
    const url = req.url ?? '';
    if (url === '/comics' || url.startsWith('/comics?')) {
      const query = url.slice('/comics'.length);
      res.statusCode = 301;
      res.setHeader('Location', `/comics/${query}`);
      res.end();
      return;
    }
    next();
  };

  return {
    name: 'comics-base-redirect',
    configureServer(server) {
      server.middlewares.use(redirect);
    },
    configurePreviewServer(server) {
      server.middlewares.use(redirect);
    },
  };
}

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
    },
  },
  build: {
    outDir: 'build',
  },
  server: {
    https: getHttpsConfig(),
    proxy: {
      '/api': {
        target: goApiTarget,
        changeOrigin: true,
        secure: false,
        rewrite: (requestPath) => requestPath.replace(/^\/api/, ''),
      },
    },
  },
  preview: {
    proxy: {
      '/api': {
        target: goApiTarget,
        changeOrigin: true,
        secure: false,
        rewrite: (requestPath) => requestPath.replace(/^\/api/, ''),
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/setupTests.ts',
  },
});

