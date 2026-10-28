import fs from 'node:fs';
import path from 'node:path';

import type { ProxyOptions } from 'vite';

const defaultTLSCert = './tls/comics.crt';
const defaultTLSKey = './tls/comics.key';

/** Pick Go listen URL for the Vite /api proxy (must match cmd/server HTTP vs HTTPS). */
export function resolveGoApiTarget(rootDir: string): string {
  const explicit = process.env.VITE_GO_API_TARGET?.trim();
  if (explicit) {
    return explicit;
  }

  const certPath = path.resolve(rootDir, process.env.SSL_CRT_FILE || defaultTLSCert);
  const keyPath = path.resolve(rootDir, process.env.SSL_KEY_FILE || defaultTLSKey);
  if (fs.existsSync(certPath) && fs.existsSync(keyPath)) {
    return 'https://127.0.0.1:8081';
  }

  return 'http://127.0.0.1:8081';
}

export function goApiDevProxy(rootDir: string = process.cwd()): Record<string, string | ProxyOptions> {
  const target = resolveGoApiTarget(rootDir);
  return {
    '/api': {
      target,
      changeOrigin: true,
      secure: false,
      rewrite: (requestPath) => requestPath.replace(/^\/api/, ''),
    },
  };
}
