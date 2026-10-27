import type { ComicListFetchArgs, ComicListFetchResult } from '../../../util/ServerHelpers';

export type CatalogQueryKeyParts = Omit<ComicListFetchArgs, 'signal' | 'server'>;

const CACHE_MAX_ENTRIES = 24;
const CACHE_TTL_MS = 2 * 60 * 1000;

type CacheEntry = {
  result: ComicListFetchResult;
  storedAt: number;
};

const cache = new Map<string, CacheEntry>();
const inflight = new Map<string, Promise<ComicListFetchResult>>();

export function buildCatalogQueryKey(parts: CatalogQueryKeyParts): string {
  return JSON.stringify({
    from: parts.from,
    limit: parts.limit,
    queryFilter: parts.queryFilter.trim(),
    onlyTracked: parts.onlyTracked,
    onlyUnchecked: parts.onlyUnchecked,
    ratingMin: parts.ratingMin ?? null,
    ratingMax: parts.ratingMax ?? null,
    sortBy: parts.sortBy ?? '',
    sortDir: parts.sortDir ?? '',
  });
}

function touchEntry(key: string, entry: CacheEntry) {
  cache.delete(key);
  cache.set(key, entry);
}

function trimCache() {
  while (cache.size > CACHE_MAX_ENTRIES) {
    const oldest = cache.keys().next().value;
    if (oldest === undefined) {
      break;
    }
    cache.delete(oldest);
  }
}

export function peekCatalogQueryCache(key: string): ComicListFetchResult | undefined {
  const entry = cache.get(key);
  if (!entry) {
    return undefined;
  }
  if (Date.now() - entry.storedAt > CACHE_TTL_MS) {
    cache.delete(key);
    return undefined;
  }
  touchEntry(key, entry);
  return entry.result;
}

export function invalidateCatalogQueryCache() {
  cache.clear();
  inflight.clear();
}

export async function fetchCatalogQuery(
  key: string,
  args: ComicListFetchArgs,
  fetcher: (args: ComicListFetchArgs) => Promise<ComicListFetchResult>,
): Promise<ComicListFetchResult> {
  const cached = peekCatalogQueryCache(key);
  if (cached) {
    return cached;
  }

  // Only dedupe prefetch/background fetches. Signed catalog loads use AbortSignal;
  // reusing an aborted in-flight promise would strand the UI on "Waking up server".
  const pending = inflight.get(key);
  if (pending && !args.signal) {
    return pending;
  }

  const promise = fetcher(args)
    .then((result) => {
      touchEntry(key, { result, storedAt: Date.now() });
      trimCache();
      return result;
    })
    .finally(() => {
      inflight.delete(key);
    });

  inflight.set(key, promise);
  return promise;
}

export function scheduleAdjacentCatalogPrefetch(
  parts: CatalogQueryKeyParts,
  fetcher: (args: ComicListFetchArgs) => Promise<ComicListFetchResult>,
) {
  if (parts.limit <= 0) {
    return;
  }

  const run = () => {
    const neighbors = [
      Math.max(0, parts.from - parts.limit),
      parts.from + parts.limit,
    ];
    for (const from of neighbors) {
      if (from === parts.from) {
        continue;
      }
      const neighborParts = { ...parts, from };
      const key = buildCatalogQueryKey(neighborParts);
      if (peekCatalogQueryCache(key)) {
        continue;
      }
      void fetchCatalogQuery(key, { ...neighborParts, signal: undefined }, fetcher).catch(() => undefined);
    }
  };

  if (typeof requestIdleCallback === 'function') {
    requestIdleCallback(run, { timeout: 1500 });
  } else {
    window.setTimeout(run, 0);
  }
}

/** Test-only reset */
export function resetCatalogQueryCacheForTests() {
  invalidateCatalogQueryCache();
}
