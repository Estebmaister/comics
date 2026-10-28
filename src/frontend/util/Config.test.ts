import { describe, expect, test } from 'vitest';
import { resolveServerURL } from './Config';

describe('resolveServerURL', () => {
  test('prefers explicit VITE_API_SERVER', () => {
    expect(resolveServerURL({
      hostname: 'localhost',
      env: {
        VITE_API_SERVER: 'https://api.example.com',
        VITE_PY_SERVER: 'http://localhost:5001',
      },
    })).toBe('https://api.example.com');
  });

  test('falls back to legacy VITE_PY_SERVER', () => {
    expect(resolveServerURL({
      hostname: 'localhost',
      env: { VITE_PY_SERVER: 'http://localhost:5001' },
    })).toBe('http://localhost:5001');
  });

  test('uses Vite /api proxy on localhost during dev', () => {
    expect(resolveServerURL({
      hostname: 'localhost',
      env: {},
      dev: true,
    })).toBe(`${window.location.origin}/api`);
  });

  test('uses Vite /api proxy for Tailscale hostname during dev', () => {
    expect(resolveServerURL({
      hostname: 'realme.persian-nominal.ts.net',
      env: {},
      dev: true,
    })).toBe(`${window.location.origin}/api`);
  });

  test('normalizes VITE_API_SERVER=/api to same-origin during dev', () => {
    expect(resolveServerURL({
      hostname: 'realme.persian-nominal.ts.net',
      env: { VITE_API_SERVER: '/api' },
      dev: true,
    })).toBe(`${window.location.origin}/api`);
  });

  test('keeps remote Python fallback for hosted frontend', () => {
    expect(resolveServerURL({
      hostname: 'comics.example.com',
      env: {},
      dev: false,
    })).toBe('https://comics.example.com:5001');
  });
});
