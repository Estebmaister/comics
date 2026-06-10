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

  test('uses local Go server by default on localhost', () => {
    expect(resolveServerURL({
      hostname: 'localhost',
      env: {},
    })).toBe('https://localhost:8081');
  });

  test('keeps remote Python fallback for hosted frontend', () => {
    expect(resolveServerURL({
      hostname: 'comics.example.com',
      env: {},
    })).toBe('https://comics.example.com:5001');
  });
});
