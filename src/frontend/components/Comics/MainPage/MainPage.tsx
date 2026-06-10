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
import { calculatePageLimit, handleSearchInput } from '../utils';
import { REFRESH_INTERVAL, SEARCH_DEBOUNCE_MS } from '../constants';
import { FloatingActionRail } from '../Actions/FloatingActionRail';
import { ToastProvider } from '../../Toast/ToastProvider';
import LoadMsgs from '../../Loaders/LoadMsgs';

export function ComicsMainPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [webComics, setWebComics] = useState<Comic[]>([]);
  const [paginationDict, setPaginationDict] = useState<PaginationState>({});
  const [loadMsg, setLoadMsg] = useState<string | JSX.Element>('');
  const [refreshTick, setRefreshTick] = useState(0);
  const [limit, setLimit] = useState(() => calculatePageLimit(window.innerWidth, window.innerHeight));
  const [isDocumentVisible, setIsDocumentVisible] = useState(() => document.visibilityState !== 'hidden');
  const [isOnline, setIsOnline] = useState(() => navigator.onLine);
  const requestIdRef = useRef(0);
  const abortControllerRef = useRef<AbortController | null>(null);
  const inFlightRef = useRef(false);

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

  const total = paginationDict.total || 1;
  const totalPages = paginationDict.totalPages || 1;
  const currentPage = paginationDict.currentPage || 1;
  const onFirstPage = from <= 0;
  const onLastPage = from >= limit * (totalPages - 1);

  const paginationData = useMemo(() => ({
    from, limit, setSearchParams,
    onFirstPage, onLastPage, currentPage, totalPages
  }), [from, limit, setSearchParams, onFirstPage, onLastPage, currentPage, totalPages]);

  const requestRefresh = useCallback(() => {
    startTransition(() => {
      setRefreshTick((value) => value + 1);
    });
  }, []);

  const handleFilteredMutationSuccess = useCallback(() => {
    // Only refill the page when both filters are active.
    if (!(onlyTracked && onlyUnchecked)) return;
    requestRefresh();
  }, [onlyTracked, onlyUnchecked, requestRefresh]);

  const handleSearchInputChange = useCallback((value: string) => {
    setSearchInput(value);
  }, []);

  const runFetch = useCallback(async ({
    silent = false,
    skipIfBusy = false,
  }: { silent?: boolean; skipIfBusy?: boolean } = {}) => {
    if (skipIfBusy && inFlightRef.current) return;

    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;
    abortControllerRef.current?.abort();

    const controller = new AbortController();
    abortControllerRef.current = controller;
    inFlightRef.current = true;
    if (!silent) setLoadMsg(LoadMsgs.wait);

    try {
      const result = await dataFetch({
        from,
        limit,
        queryFilter,
        onlyTracked,
        onlyUnchecked,
        ratingMin,
        ratingMax,
        sortBy,
        sortDir,
        signal: controller.signal,
      });
      if (controller.signal.aborted || requestId !== requestIdRef.current) return;

      startTransition(() => {
        setWebComics(result.comics);
        setPaginationDict(result.pagination);
        setLoadMsg('');
      });
    } catch (error) {
      if (isAbortError(error) || requestId !== requestIdRef.current) return;
      if (silent) return;

      setWebComics([]);
      setLoadMsg(
        error instanceof ComicListFetchError && error.kind === 'server'
          ? LoadMsgs.server
          : LoadMsgs.network
      );
      console.debug((error as Error)?.message ?? error);
    } finally {
      if (requestId === requestIdRef.current) {
        abortControllerRef.current = null;
        inFlightRef.current = false;
      }
    }
  }, [from, limit, queryFilter, onlyTracked, onlyUnchecked, ratingMax, ratingMin, sortBy, sortDir]);

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
    const handleResize = () => {
      cancelAnimationFrame(rafId);
      rafId = window.requestAnimationFrame(() => {
        const nextLimit = calculatePageLimit(window.innerWidth, window.innerHeight);
        setLimit((currentLimit) => currentLimit === nextLimit ? currentLimit : nextLimit);
      });
    };

    window.addEventListener('resize', handleResize);
    return () => {
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    void runFetch();

    return () => {
      abortControllerRef.current?.abort();
    };
  }, [runFetch, refreshTick]);

  useEffect(() => {
    if (!isDocumentVisible || !isOnline || isSearchActive) return undefined;

    const intervalId = window.setInterval(() => {
      void runFetch({ silent: true, skipIfBusy: true });
    }, REFRESH_INTERVAL);

    return () => window.clearInterval(intervalId);
  }, [isDocumentVisible, isOnline, isSearchActive, runFetch]);

  useEffect(() => {
    // Edge case: after checkout on the last page (tracked + unchecked),
    // page can become empty. Jump to the last valid page offset.
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
      <main className="app-shell min-h-screen pb-24">
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
        />

        <ComicsList
          comics={webComics}
          loadMsg={loadMsg}
          queryFilter={queryFilter}
          onCheckoutSuccess={handleFilteredMutationSuccess}
          onDeleteSuccess={handleFilteredMutationSuccess}
        />

        <FloatingActionRail>
          <ScrapeButton onSuccess={requestRefresh} />
          <MergeComic onSuccess={requestRefresh} />
          <CreateComic onSuccess={requestRefresh} />
        </FloatingActionRail>
      </main>
    </ToastProvider>
  );
}
