import type { SetStateAction } from "react";
import config from "./Config";
import type { Comic, PaginationState } from "../components/Comics/types";

const SERVER = config.SERVER;

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
  console.debug(url);

  let response: Response;
  try {
    response = await fetch(url, {
      method: "GET",
      headers: { accept: "application/json" },
    });
  } catch (error) {
    if (isAbortError(error)) throw error;
    throw new ComicListFetchError((error as Error)?.message ?? 'Network request failed');
  }

  console.debug(response);
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

  console.debug("Response succeed", data);
  return {
    comics: data,
    pagination: {
      total: Number(response.headers.get("total-comics") || 0),
      totalPages: Number(response.headers.get("total-pages") || 1),
      currentPage: Number(response.headers.get("current-page") || 1),
    },
  };
};

const trackComic = (
  tracked: boolean,
  id: number,
  setTrack: (value: boolean) => void,
  server = SERVER,
) => {
  fetch(`${server}/comics/${id}`, {
    method: "PUT",
    body: JSON.stringify({ track: !tracked }),
    headers: { "Content-Type": "application/json" },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      setTrack(!tracked);
    })
    .catch((err) => {
      console.debug(err.message);
    });
};

const rateComic = async (
  rating: number,
  id: number,
  setComic: (value: SetStateAction<Comic>) => void,
  server = SERVER,
): Promise<boolean> => {
  const next = Math.max(0, Math.min(5, Number.isFinite(rating) ? rating : 0));
  try {
    const response = await fetch(`${server}/comics/${id}`, {
      method: "PUT",
      body: JSON.stringify({ rating: next }),
      headers: { "Content-Type": "application/json" },
    });
    const data = await response.json();
    console.debug(data);
    if (!response.ok || data?.message !== undefined) return false;
    setComic((prev) => ({ ...prev, rating: next }));
    return true;
  } catch (err) {
    console.debug((err as Error)?.message ?? err);
    return false;
  }
};

const checkoutComic = (
  curr_chap: number,
  id: number,
  setCheck: (value: SetStateAction<boolean>) => void,
  setViewedChap: (value: SetStateAction<number>) => void,
  onSuccess?: () => void,
  server = SERVER,
) => {
  fetch(`${server}/comics/${id}`, {
    method: "PUT",
    body: JSON.stringify({ viewed_chap: curr_chap }),
    headers: { "Content-Type": "application/json" },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      setCheck(false);
      setViewedChap(curr_chap);
      onSuccess?.();
    })
    .catch((err) => {
      console.debug(err.message);
    });
};

const delComic = (
  id: number,
  setDelete: (value: SetStateAction<boolean>) => void,
  onSuccess?: () => void,
  server = SERVER,
) => {
  fetch(`${server}/comics/${id}`, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      setDelete(true);
      onSuccess?.();
    })
    .catch((err) => {
      console.debug(err.message);
    });
};

const reportCoverVisibility = async (
  id: number,
  cover: string,
  cover_visible: boolean,
  server = SERVER,
): Promise<Comic | undefined> => {
  const response = await fetch(`${server}/comics/${id}/cover-visibility`, {
    method: "PATCH",
    body: JSON.stringify({ cover, cover_visible }),
    headers: { "Content-Type": "application/json" },
  });
  const data = await response.json();
  if (!response.ok || data?.message !== undefined) {
    throw new Error(data?.message ?? `Cover visibility update failed (${response.status})`);
  }
  return data;
};

export { dataFetch, trackComic, rateComic, checkoutComic, delComic, reportCoverVisibility, isAbortError };
