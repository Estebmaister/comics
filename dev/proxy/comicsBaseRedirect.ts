import type { Connect } from 'vite';
import type { Plugin } from 'vite';

/** Redirect /comics → /comics/ so React Router basename matches. */
export function comicsBaseRedirectPlugin(): Plugin {
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
