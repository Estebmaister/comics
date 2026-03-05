import { JSX, useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../../../css/main.css';

import { NavBar } from '../Navigation/NavBar';
import ComicsList from '../Card/ComicsList';
import CreateComic from '../Edition/CreateComic';
import MergeComic from '../Edition/MergeComic';
import ScrapeButton from '../Edition/ScrapeButton';
import { dataFetch } from '../../../util/ServerHelpers';
import { Comic, PaginationState } from '../types';
import { calculatePageLimit } from '../utils';
import { REFRESH_INTERVAL } from '../constants';

export function ComicsMainPage() {
  // URL parameters and state
  const [searchParams, setSearchParams] = useSearchParams();
  const [webComics, setWebComics] = useState<Comic[]>([]);
  const [paginationDict, setPaginationDict] = useState<PaginationState>({});
  const [loadMsg, setLoadMsg] = useState<string | JSX.Element>('');
  const [refreshTick, setRefreshTick] = useState(0);
  const [viewport, setViewport] = useState(() => ({
    width: window.innerWidth,
    height: window.innerHeight,
  }));

  // Parse URL parameters
  const onlyUnchecked = searchParams.get('onlyUnchecked') === 'true';
  const onlyTracked = searchParams.get('onlyTracked') === 'true';
  const queryFilter = searchParams.get('queryFilter') || '';
  const from = parseInt(searchParams.get('from') || '0') || 0;
  const limit = calculatePageLimit(viewport.width, viewport.height);

  // Calculate pagination state
  const total = paginationDict.total || 1;
  const totalPages = paginationDict.totalPages || 1;
  const currentPage = paginationDict.currentPage || 1;
  const onFirstPage = from <= 0;
  const onLastPage = from >= limit * (totalPages - 1);

  const paginationData = {
    from, limit, setSearchParams,
    onFirstPage, onLastPage, currentPage, totalPages
  };

  const handleFilteredMutationSuccess = useCallback(() => {
    // Only refill the page when both filters are active.
    if (!(onlyTracked && onlyUnchecked)) return;
    setRefreshTick((value) => value + 1);
  }, [onlyTracked, onlyUnchecked]);

  useEffect(() => {
    let rafId = 0;
    const handleResize = () => {
      cancelAnimationFrame(rafId);
      rafId = window.requestAnimationFrame(() => {
        setViewport({
          width: window.innerWidth,
          height: window.innerHeight,
        });
      });
    };

    window.addEventListener('resize', handleResize);
    return () => {
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    const fetchData = () => {
      dataFetch(
        { setWebComics, setPaginationDict, setLoadMsg },
        from, limit, queryFilter, onlyTracked, onlyUnchecked
      );
    };

    // Initial fetch
    fetchData();

    // Set up interval for periodic fetches
    const intervalId = setInterval(fetchData, REFRESH_INTERVAL);

    // Cleanup on unmount
    return () => clearInterval(intervalId);
  }, [from, limit, queryFilter, onlyTracked, onlyUnchecked, refreshTick]);

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
      prev.set('from', String(nextFrom));
      return prev;
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
    <main className="min-h-screen pb-24">
      <NavBar
        onlyTracked={onlyTracked}
        onlyUnchecked={onlyUnchecked}
        total={total}
        queryFilter={queryFilter}
        setSearchParams={setSearchParams}
        paginationData={paginationData}
      />

      <ComicsList
        comics={webComics}
        loadMsg={loadMsg}
        queryFilter={queryFilter}
        onCheckoutSuccess={handleFilteredMutationSuccess}
        onDeleteSuccess={handleFilteredMutationSuccess}
      />

      <MergeComic />
      <CreateComic />
      <ScrapeButton />
    </main>
  );
}
