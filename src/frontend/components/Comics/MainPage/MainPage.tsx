import { startTransition, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { JSX } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../../../css/main.css';

import { NavBar } from '../Navigation/NavBar';
import ComicsList from '../Card/ComicsList';
import CreateComic from '../Edition/CreateComic';
import MergeComic from '../Edition/MergeComic';
import ScrapeButton from '../Edition/ScrapeButton';
import { ComicListFetchError, dataFetch, isAbortError } from '../../../util/ServerHelpers';
import { Comic, PaginationState } from '../types';
import { NAV_HEIGHT_UPDATED_EVENT } from '../../../hooks/useMeasuredNavHeight';
import { calculatePageLimit, handleSearchInput, readNavHeightPx } from '../utils';
import { REFRESH_INTERVAL, SEARCH_DEBOUNCE_MS } from '../constants';
import { FloatingActionRail } from '../Actions/FloatingActionRail';
import { ToastProvider } from '../../Toast/ToastProvider';
import { DiscoveryConflictProvider } from '../Discovery/DiscoveryConflictContext';
import CatalogDuplicateWatcher from '../Discovery/CatalogDuplicateWatcher';
import LoadMsgs from '../../Loaders/LoadMsgs';
import { useAuth } from '../../../context/AuthContext';
import {
  buildCatalogQueryKey,
  fetchCatalogQuery,
  invalidateCatalogQueryCache,
  peekCatalogQueryCache,
  scheduleAdjacentCatalogPrefetch,
} from './catalogQueryCache';

export function ComicsMainPage() {
  const { isAdmin } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [webComics, setWebComics] = useState<Comic[]>([]);
  const [paginationDict, setPaginationDict] = useState<PaginationState>({});
  const [loadMsg, setLoadMsg] = useState<string | JSX.Element>('');
  const [refreshTick, setRefreshTick] = useState(0);
  const [limit, setLimit] = useState(() => calculatePageLimit(
    window.innerWidth,
    window.innerHeight,
    { navHeightPx: readNavHeightPx() },
  ));
  const [isDocumentVisible, setIsDocumentVisible] = useState(() => document.visibilityState !== 'hidden');
  const [isOnline, setIsOnline] = useState(() => navigator.onLine);
  const [isFetching, setIsFetching] = useState(false);
  const [displayedQueryKey, setDisplayedQueryKey] = useState('');
  const [fetchError, setFetchError] = useState<'network' | 'server' | null>(null);
  const requestIdRef = useRef(0);
  const abortControllerRef = useRef<AbortController | null>(null);
  const inFlightRef = useRef(false);
  const hasDisplayedComicsRef = useRef(false);

  const onlyUnchecked = searchParams.get('onlyUnchecked') === 'true';
  const onlyTracked = searchParams.get('onlyTracked') === 'true';
  const queryFilter = searchParams.get('queryFilter') || '';
  const [searchInput, setSearchInput] = useState(() => queryFilter);
  const lastQueryFilterRef = useRef(queryFilter);
  const from = parseInt(searchParams.get('from') || '0') || 0;
  const isSearchActive = searchInput !== queryFilter;
  const ratingMin = (() => {
    const raw = searchParams.get('ratingMin');
    if (!raw) return undefined;
    const parsed = Number(raw);
    if (!Number.isFinite(parsed)) return undefined;
    const value = Math.trunc(parsed);
    return value >= 0 && value <= 5 ? value : undefined;
  })();
  const ratingMax = (() => {
    const raw = searchParams.get('ratingMax');
    if (!raw) return undefined;
    const parsed = Number(raw);
    if (!Number.isFinite(parsed)) return undefined;
    const value = Math.trunc(parsed);
    return value >= 0 && value <= 5 ? value : undefined;
  })();
  const sortBy = searchParams.get('sortBy') || undefined;
  const sortDir = searchParams.get('sortDir') || undefined;

  const queryKeyParts = useMemo(() => ({
    from,
    limit,
    queryFilter,
    onlyTracked,
    onlyUnchecked,
    ratingMin,
    ratingMax,
    sortBy,
    sortDir,
  }), [from, limit, queryFilter, onlyTracked, onlyUnchecked, ratingMax, ratingMin, sortBy, sortDir]);

  const requestedQueryKey = useMemo(
    () => buildCatalogQueryKey(queryKeyParts),
    [queryKeyParts],
  );

  const targetPage = limit > 0 ? Math.floor(from / limit) + 1 : 1;
  const total = paginationDict.total || 1;
  const totalPages = paginationDict.totalPages || 1;
  const settledCurrentPage = paginationDict.currentPage || targetPage;
  const currentPage = displayedQueryKey !== requestedQueryKey || isFetching
    ? targetPage
    : settledCurrentPage;
  const onFirstPage = from <= 0;
  const onLastPage = from >= limit * (totalPages - 1);
  const isShowingStaleResults = webComics.length > 0 && displayedQueryKey !== requestedQueryKey;
  const isCatalogPending = isFetching || isSearchActive || isShowingStaleResults;

  const paginationData = useMemo(() => ({
    from, limit, setSearchParams,
    onFirstPage, onLastPage, currentPage, totalPages,
    isPending: isCatalogPending,
  }), [from, limit, setSearchParams, onFirstPage, onLastPage, currentPage, totalPages, isCatalogPending]);

  const requestRefresh = useCallback((removedComicIds: number[] = []) => {
    invalidateCatalogQueryCache();
    if (removedComicIds.length > 0) {
      const removed = new Set(removedComicIds);
      startTransition(() => {
        setWebComics((prev) => prev.filter((comic) => !removed.has(comic.id)));
        setPaginationDict((prev) => ({
          ...prev,
          total: Math.max(0, (prev.total ?? 0) - removed.size),
        }));
      });
    }
    startTransition(() => {
      setRefreshTick((value) => value + 1);
    });
    queueMicrotask(() => {
      void runFetchRef.current();
    });
  }, []);

  const handleFilteredMutationSuccess = useCallback(() => {
    if (!(onlyTracked && onlyUnchecked)) return;
    requestRefresh();
  }, [onlyTracked, onlyUnchecked, requestRefresh]);

  const handleSearchInputChange = useCallback((value: string) => {
    setSearchInput(value);
  }, []);

  const applyListResult = useCallback((result: { comics: Comic[]; pagination: PaginationState }, key: string) => {
    hasDisplayedComicsRef.current = result.comics.length > 0;
    startTransition(() => {
      setWebComics(result.comics);
      setPaginationDict(result.pagination);
      setDisplayedQueryKey(key);
      setFetchError(null);
      setLoadMsg('');
    });
  }, []);

  const runFetch = useCallback(async ({
    silent = false,
    skipIfBusy = false,
  }: { silent?: boolean; skipIfBusy?: boolean } = {}) => {
    if (skipIfBusy && inFlightRef.current) return;

    const key = requestedQueryKey;
    const cached = peekCatalogQueryCache(key);
    if (cached) {
      applyListResult(cached, key);
      scheduleAdjacentCatalogPrefetch(queryKeyParts, dataFetch);
      return;
    }

    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;

    const controller = new AbortController();
    abortControllerRef.current = controller;
    inFlightRef.current = true;
    setIsFetching(true);
    if (!silent && !hasDisplayedComicsRef.current && !peekCatalogQueryCache(key)) {
      setLoadMsg(LoadMsgs.wait);
    }

    try {
      const result = await fetchCatalogQuery(key, {
        ...queryKeyParts,
        signal: controller.signal,
      }, dataFetch);
      if (controller.signal.aborted || requestId !== requestIdRef.current) return;

      applyListResult(result, key);
      scheduleAdjacentCatalogPrefetch(queryKeyParts, dataFetch);
    } catch (error) {
      if (isAbortError(error) || requestId !== requestIdRef.current) return;
      if (silent) return;

      const kind = error instanceof ComicListFetchError && error.kind === 'server' ? 'server' : 'network';
      setFetchError(kind);
      if (!hasDisplayedComicsRef.current) {
        setWebComics([]);
        setLoadMsg(kind === 'server' ? LoadMsgs.server : LoadMsgs.network);
      } else {
        setLoadMsg('');
      }
      console.debug((error as Error)?.message ?? error);
    } finally {
      inFlightRef.current = false;
      if (requestId === requestIdRef.current) {
        abortControllerRef.current = null;
        setIsFetching(false);
      }
    }
  }, [applyListResult, queryKeyParts, requestedQueryKey]);

  const runFetchRef = useRef(runFetch);
  runFetchRef.current = runFetch;

  useEffect(() => {
    if (queryFilter === lastQueryFilterRef.current) return;
    lastQueryFilterRef.current = queryFilter;
    setSearchInput(queryFilter);
  }, [queryFilter]);

  useEffect(() => {
    if (searchInput === queryFilter) return undefined;

    const timeoutId = window.setTimeout(() => {
      handleSearchInput(setSearchParams, searchInput);
    }, SEARCH_DEBOUNCE_MS);

    return () => window.clearTimeout(timeoutId);
  }, [searchInput, queryFilter, setSearchParams]);

  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsDocumentVisible(document.visibilityState !== 'hidden');
    };
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  useEffect(() => {
    let rafId = 0;
    const syncPageLimit = () => {
      cancelAnimationFrame(rafId);
      rafId = window.requestAnimationFrame(() => {
        const nextLimit = calculatePageLimit(window.innerWidth, window.innerHeight);
        setLimit((currentLimit) => currentLimit === nextLimit ? currentLimit : nextLimit);
      });
    };

    window.addEventListener('resize', syncPageLimit);
    window.addEventListener(NAV_HEIGHT_UPDATED_EVENT, syncPageLimit);
    return () => {
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', syncPageLimit);
      window.removeEventListener(NAV_HEIGHT_UPDATED_EVENT, syncPageLimit);
    };
  }, []);

  useEffect(() => {
    void runFetchRef.current();
  }, [requestedQueryKey, refreshTick]);

  useEffect(() => {
    if (!isDocumentVisible || !isOnline || isSearchActive) return undefined;

    const intervalId = window.setInterval(() => {
      void runFetch({ silent: true, skipIfBusy: true });
    }, REFRESH_INTERVAL);

    return () => window.clearInterval(intervalId);
  }, [isDocumentVisible, isOnline, isSearchActive, runFetch]);

  useEffect(() => {
    if (!(onlyTracked && onlyUnchecked)) return;
    if (paginationDict.totalPages === undefined) return;
    if (from <= 0 || webComics.length > 0) return;
    const validTotalPages = Math.max(1, Number(paginationDict.totalPages) || 1);
    const nextFrom = Math.max(0, limit * (validTotalPages - 1));
    if (nextFrom === from) return;
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('from', String(nextFrom));
      return next;
    }, { replace: false });
  }, [
    onlyTracked,
    onlyUnchecked,
    paginationDict.totalPages,
    from,
    limit,
    webComics.length,
    setSearchParams,
  ]);

  return (
    <ToastProvider>
      <DiscoveryConflictProvider
        onCatalogRefresh={(removedIds) => requestRefresh(removedIds)}
      >
      <main className="app-shell">
        <NavBar
          onlyTracked={onlyTracked}
          onlyUnchecked={onlyUnchecked}
          total={total}
          queryFilter={searchInput}
          onQueryFilterChange={handleSearchInputChange}
          setSearchParams={setSearchParams}
          paginationData={paginationData}
          ratingMin={ratingMin}
          ratingMax={ratingMax}
          sortBy={sortBy}
          sortDir={sortDir}
          isCatalogPending={isCatalogPending}
        />

        <ComicsList
          comics={webComics}
          loadMsg={loadMsg}
          queryFilter={queryFilter}
          isFetching={isCatalogPending}
          isShowingStaleResults={isShowingStaleResults}
          fetchError={fetchError}
          onCheckoutSuccess={handleFilteredMutationSuccess}
          onDeleteSuccess={handleFilteredMutationSuccess}
        />

        {isAdmin ? (
          <CatalogDuplicateWatcher
            comics={webComics}
            enabled={!isCatalogPending && displayedQueryKey === requestedQueryKey}
            viewKey={displayedQueryKey}
          />
        ) : null}

        {isAdmin ? (
          <FloatingActionRail>
            <ScrapeButton onSuccess={requestRefresh} />
            <MergeComic
              onSuccess={({ mergingID }) => requestRefresh([mergingID])}
            />
            <CreateComic onSuccess={requestRefresh} />
          </FloatingActionRail>
        ) : null}
      </main>
      </DiscoveryConflictProvider>
    </ToastProvider>
  );
}
