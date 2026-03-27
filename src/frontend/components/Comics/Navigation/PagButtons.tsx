import { SetURLSearchParams } from 'react-router-dom';
import { PaginationData } from '../types';

const handlePagination = (
  direction: string,
  { from, limit, totalPages, setSearchParams }:
    { from: number, limit: number, totalPages: number, setSearchParams: SetURLSearchParams }
) => {
  const LAST_FROM = limit * (totalPages - 1)
  let moveFrom = 0
  switch (direction) {
    case 'next':
      moveFrom = +limit;
      break;
    case 'prev':
      moveFrom = -limit;
      break;
    case 'first':
      moveFrom = -from;
      break;
    case 'last':
      moveFrom = -from + LAST_FROM;
      break;
    default:
      console.error('Pagination called without valid argument: ', direction);
      return;
  };
  // Border cases, before first page, after last page
  if (from + moveFrom < 0) moveFrom = -from;
  else if (from + moveFrom > LAST_FROM) moveFrom = -from + LAST_FROM;

  setSearchParams((prev) => {
    prev.set('from', String(from + moveFrom));
    if (from + moveFrom === 0) prev.delete('from');
    return prev;
  }, { replace: false });
}

export default function PagButtons({ pagD }: { pagD: PaginationData }) {
  return (
    <div className="grid w-full grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_minmax(0,1fr)_minmax(0,1fr)] items-center gap-1.5 sm:ml-auto sm:flex sm:w-auto sm:justify-end sm:gap-1">
      <button
        className="basic-button reverse-button min-h-[2.35rem] min-w-0 px-2.5 text-[0.7rem] leading-none sm:min-w-[4.2rem] sm:px-3 sm:text-[0.72rem]"
        disabled={pagD.onFirstPage}
        onClick={() => handlePagination('first', pagD)}
      >
        First
      </button>
      <button
        className="basic-button reverse-button min-h-[2.35rem] min-w-0 px-2.5 text-[0.7rem] leading-none sm:min-w-[4.2rem] sm:px-3 sm:text-[0.72rem]"
        disabled={pagD.onFirstPage}
        onClick={() => handlePagination('prev', pagD)}
      >
        Prev
      </button>
      <span className="page-indicator min-w-[2.3rem] px-2 py-1 text-[0.82rem] font-semibold sm:min-w-[2.7rem] sm:text-sm">
        {pagD.currentPage}
      </span>
      <button
        className="basic-button min-h-[2.35rem] min-w-0 px-2.5 text-[0.7rem] leading-none sm:min-w-[4.2rem] sm:px-3 sm:text-[0.72rem]"
        disabled={pagD.onLastPage}
        onClick={() => handlePagination('next', pagD)}
      >
        Next
      </button>
      <button
        className="basic-button min-h-[2.35rem] min-w-0 px-2.5 text-[0.7rem] leading-none sm:min-w-[5.6rem] sm:px-3 sm:text-[0.72rem]"
        disabled={pagD.onLastPage}
        onClick={() => handlePagination('last', pagD)}
      >
        Last ({pagD.totalPages})
      </button>
    </div>
  );
};
