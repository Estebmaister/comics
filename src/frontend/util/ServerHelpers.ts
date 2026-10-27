import type { SetStateAction } from "react";
import { apiErrorMessage, authorizedFetch } from "../api/client";
import config from "./Config";
import type { Comic, PaginationState } from "../components/Comics/types";
import type { CatalogConflictPayload } from "./catalogConflict";
import { patchComic } from "./comicUpdate";

const SERVER = config.SERVER;

export type CatalogMutationOptions = {
  onConflict?: (
    conflict: CatalogConflictPayload,
    retry: () => Promise<boolean>,
  ) => void;
};

export type ComicListFetchArgs = {
  from: number;
  limit: number;
  queryFilter: string;
  onlyTracked: boolean;
  onlyUnchecked: boolean;
  ratingMin?: number;
  ratingMax?: number;
  sortBy?: string;
  sortDir?: string;
  signal?: AbortSignal;
  server?: string;
};

export type ComicListFetchResult = {
  comics: Comic[];
  pagination: PaginationState;
};

export class ComicListFetchError extends Error {
  constructor(
    message: string,
    public readonly kind: 'network' | 'server' = 'network',
  ) {
    super(message);
    this.name = 'ComicListFetchError';
  }
}

const isAbortError = (error: unknown) => (
  error !== null
  && typeof error === 'object'
  && 'name' in error
  && (error as { name?: string }).name === 'AbortError'
);

const dataFetch = async ({
  from,
  limit,
  queryFilter,
  onlyTracked,
  onlyUnchecked,
  ratingMin,
  ratingMax,
  sortBy,
  sortDir,
  signal,
  server = SERVER,
}: ComicListFetchArgs): Promise<ComicListFetchResult> => {
  const trimmedFilter = queryFilter.trim();
  const baseURL = trimmedFilter === ''
    ? `${server}/comics`
    : `${server}/comics/search/${encodeURIComponent(trimmedFilter)}`;
  const params = new URLSearchParams({
    from: String(from),
    limit: String(limit),
    only_tracked: String(onlyTracked),
    only_unchecked: String(onlyUnchecked),
  });
  if (ratingMin !== undefined) params.set('rating_min', String(ratingMin));
  if (ratingMax !== undefined) params.set('rating_max', String(ratingMax));
  if (sortBy) params.set('sort_by', sortBy);
  if (sortDir) params.set('sort_dir', sortDir);
  const url = `${baseURL}?${params.toString()}`;

  let response: Response;
  try {
    response = await fetch(url, {
      method: "GET",
      headers: { accept: "application/json" },
      signal,
    });
  } catch (error) {
    if (isAbortError(error)) throw error;
    throw new ComicListFetchError((error as Error)?.message ?? 'Network request failed');
  }

  if (signal?.aborted) {
    throw new DOMException('The operation was aborted', 'AbortError');
  }

  const data = await response.json();
  if (!response.ok || data?.message !== undefined) {
    throw new ComicListFetchError(
      data?.message ?? `Server request failed (${response.status})`,
      'server',
    );
  }
  if (!Array.isArray(data)) {
    throw new ComicListFetchError('Server returned an unexpected comics payload', 'server');
  }

  return {
    comics: data,
    pagination: {
      total: Number(response.headers.get("total-comics") || 0),
      totalPages: Number(response.headers.get("total-pages") || 1),
      currentPage: Number(response.headers.get("current-page") || 1),
    },
  };
};

const applyComicPatchWithConflict = async (
  comicId: number,
  body: Record<string, unknown>,
  options?: CatalogMutationOptions,
  server = SERVER,
): Promise<boolean> => {
  const attempt = async (): Promise<boolean> => {
    const result = await patchComic(comicId, body, server);
    if (result.ok) return true;
    if (result.kind === 'conflict' && options?.onConflict) {
      options.onConflict(result.conflict, attempt);
      return false;
    }
    if (result.kind === 'server' || result.kind === 'network') {
      console.debug(result.kind === 'server' ? result.message : 'network error');
    }
    return false;
  };
  return attempt();
};

const trackComic = async (
  tracked: boolean,
  id: number,
  setTrack: (value: boolean) => void,
  options?: CatalogMutationOptions,
  server = SERVER,
): Promise<boolean> => {
  const ok = await applyComicPatchWithConflict(
    id,
    { track: !tracked },
    options,
    server,
  );
  if (!ok) return false;
  setTrack(!tracked);
  return true;
};

const rateComic = async (
  rating: number,
  id: number,
  setComic: (value: SetStateAction<Comic>) => void,
  options?: CatalogMutationOptions,
  server = SERVER,
): Promise<boolean> => {
  const next = Math.max(0, Math.min(5, Number.isFinite(rating) ? rating : 0));
  const ok = await applyComicPatchWithConflict(id, { rating: next }, options, server);
  if (!ok) return false;
  setComic((prev) => ({ ...prev, rating: next }));
  return true;
};

const checkoutComic = async (
  currChap: number,
  id: number,
  setCheck: (value: SetStateAction<boolean>) => void,
  setViewedChap: (value: SetStateAction<number>) => void,
  onSuccess?: () => void,
  options?: CatalogMutationOptions,
  server = SERVER,
): Promise<boolean> => {
  const ok = await applyComicPatchWithConflict(
    id,
    { viewed_chap: currChap },
    options,
    server,
  );
  if (!ok) return false;
  setCheck(false);
  setViewedChap(currChap);
  onSuccess?.();
  return true;
};

const delComic = async (
  id: number,
  setDelete: (value: SetStateAction<boolean>) => void,
  onSuccess?: () => void,
  server = SERVER,
): Promise<boolean> => {
  try {
    const response = await authorizedFetch(
      `/comics/${id}`,
      { method: "DELETE" },
      server,
    );
    await response.json().catch(() => null);
    if (!response.ok) return false;
    setDelete(true);
    onSuccess?.();
    return true;
  } catch (error) {
    console.debug((error as Error)?.message ?? error);
    return false;
  }
};

const reportCoverVisibility = async (
  id: number,
  cover: string,
  cover_visible: boolean,
  server = SERVER,
): Promise<Comic | undefined> => {
  const response = await authorizedFetch(
    `/comics/${id}/cover-visibility`,
    {
      method: "PATCH",
      body: JSON.stringify({ cover, cover_visible }),
    },
    server,
  );
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(apiErrorMessage(
      data,
      `Cover visibility update failed (${response.status})`,
    ));
  }
  return data as Comic;
};

export { dataFetch, trackComic, rateComic, checkoutComic, delComic, reportCoverVisibility, isAbortError };
