const rawBase = import.meta.env.BASE_URL ?? '/';

export const appBasename = rawBase.endsWith('/') && rawBase.length > 1
  ? rawBase.slice(0, -1)
  : rawBase === '/'
    ? undefined
    : rawBase;
