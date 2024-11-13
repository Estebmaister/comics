const handlePagination = (
    direction: string, { from, limit, totalPages, setSearchParams }: any
  ) => {
  const LAST_FROM = limit*(totalPages-1)
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
  if      (from + moveFrom < 0        ) moveFrom = -from;
  else if (from + moveFrom > LAST_FROM) moveFrom = -from + LAST_FROM;
  
  setSearchParams((prev: { set: (arg0: string, arg1: any) => void; delete: (arg0: string) => void; }) => {
    prev.set('from', from + moveFrom);
    if (from + moveFrom === 0) prev.delete('from');
    return prev;
  }, {replace: true});
}

export default function PaginationButtons({pagD}: any) {
  return (<div className='div-pagination-buttons'>
    <button className={'basic-button bar-button reverse-button' + 
        (pagD.onFirstPage ? ' disabled-button' : '')}
      disabled={pagD.onFirstPage} onClick={() =>handlePagination('first',pagD)}>
        First
    </button>
    <button className={'basic-button bar-button reverse-button' + 
        (pagD.onFirstPage ? ' disabled-button' : '')} 
      disabled={pagD.onFirstPage} onClick={() => handlePagination('prev',pagD)}>
        Prev
    </button>
    <button className={'pag-button'}> {pagD.currentPage} </button>
    <button className={'basic-button bar-button' +
        (pagD.onLastPage ? ' disabled-button' : '')}
      disabled={pagD.onLastPage} onClick={() => handlePagination('next',pagD)}>
        Next
    </button>
    <button className={'basic-button bar-button' +
        (pagD.onLastPage ? ' disabled-button' : '')}
      disabled={pagD.onLastPage} onClick={() => handlePagination('last',pagD)}>
        Last ({pagD.totalPages})
    </button>
  </div>);
};
