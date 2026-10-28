import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, test } from 'vitest';

import { resolveGoApiTarget } from './viteProxy';

describe('resolveGoApiTarget', () => {
  const originalEnv = process.env.VITE_GO_API_TARGET;

  afterEach(() => {
    if (originalEnv === undefined) {
      delete process.env.VITE_GO_API_TARGET;
    } else {
      process.env.VITE_GO_API_TARGET = originalEnv;
    }
  });

  test('prefers VITE_GO_API_TARGET when set', () => {
    process.env.VITE_GO_API_TARGET = 'http://127.0.0.1:9090';
    expect(resolveGoApiTarget('/any')).toBe('http://127.0.0.1:9090');
  });

  test('uses http when TLS files are missing', () => {
    delete process.env.VITE_GO_API_TARGET;
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'comics-vite-proxy-'));
    expect(resolveGoApiTarget(dir)).toBe('http://127.0.0.1:8081');
  });

  test('uses https when repo TLS files exist', () => {
    delete process.env.VITE_GO_API_TARGET;
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'comics-vite-proxy-'));
    fs.mkdirSync(path.join(dir, 'tls'), { recursive: true });
    fs.writeFileSync(path.join(dir, 'tls', 'comics.crt'), 'x');
    fs.writeFileSync(path.join(dir, 'tls', 'comics.key'), 'y');
    expect(resolveGoApiTarget(dir)).toBe('https://127.0.0.1:8081');
  });
});
