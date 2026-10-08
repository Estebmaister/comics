import type { ProxyOptions } from 'vite';

/** Go API origin for local dev; not used in production builds. */
export const goApiTarget =
  process.env.VITE_GO_API_TARGET ?? 'https://localhost:8081';

export function goApiDevProxy(): Record<string, string | ProxyOptions> {
  return {
    '/api': {
      target: goApiTarget,
      changeOrigin: true,
      secure: false,
      rewrite: (requestPath) => requestPath.replace(/^\/api/, ''),
    },
  };
}
