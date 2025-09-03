import { JSX, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../../../css/main.css';

import { NavBar } from '../Navigation/NavBar';
import ComicsList from '../Card/ComicsList';
import CreateComic from '../Edition/CreateComic';
import MergeComic from '../Edition/MergeComic';
import ScrapeButton from '../Edition/ScrapeButton';
import { dataFetch } from '../../../util/ServerHelpers';
import { Comic, PaginationState } from '../types';
import { calculateInlineComics } from '../utils';
import { COMICS_PER_ROW, REFRESH_INTERVAL } from '../constants';

export function ComicsMainPage() {
  // URL parameters and state
  const [searchParams, setSearchParams] = useSearchParams();
  const [webComics, setWebComics] = useState<Comic[]>([]);
  const [paginationDict, setPaginationDict] = useState<PaginationState>({});
  const [loadMsg, setLoadMsg] = useState<string | JSX.Element>('');

  // Parse URL parameters
  const onlyUnchecked = searchParams.get('onlyUnchecked') === 'true';
  const onlyTracked = searchParams.get('onlyTracked') === 'true';
  const queryFilter = searchParams.get('queryFilter') || '';
  const from = parseInt(searchParams.get('from') || '0') || 0;
  const limit = calculateInlineComics() * COMICS_PER_ROW;

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
  }, [from, limit, queryFilter, onlyTracked, onlyUnchecked]);

  return (
    <>
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
      />

      <MergeComic />
      <CreateComic />
      <ScrapeButton />
    </>
  );
}