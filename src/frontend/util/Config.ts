type ApiConfigEnv = {
  VITE_API_SERVER?: string;
  VITE_PY_SERVER?: string;
};

type ResolveServerArgs = {
  hostname?: string;
  env?: ApiConfigEnv;
};

const localHosts = new Set(['localhost', '127.0.0.1', '::1']);

const cleanURL = (value?: string) => {
  const trimmed = value?.trim();
  return trimmed === '' ? undefined : trimmed;
};

export const resolveServerURL = ({
  hostname = window.location.hostname,
  env = import.meta.env,
}: ResolveServerArgs = {}) => {
  const explicit = cleanURL(env.VITE_API_SERVER) ?? cleanURL(env.VITE_PY_SERVER);
  if (explicit) {
    return explicit;
  }
  // Vite dev: same-origin /api proxy (see dev/proxy/viteProxy.ts) avoids http→https:8081 hangs.
  if (import.meta.env.DEV && localHosts.has(hostname) && typeof window !== 'undefined') {
    return `${window.location.origin}/api`;
  }
  return localHosts.has(hostname) ? 'https://localhost:8081' : `https://${hostname}:5001`;
};

const config = {
  SHOW_MESSAGE_TIMEOUT: 2000, // Time in ms to show message
  SERVER: resolveServerURL(),
};

export default config;
