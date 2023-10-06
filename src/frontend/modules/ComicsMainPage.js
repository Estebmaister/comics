import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../css/main.css';
import { ComicCard } from './ComicCard';

export const COMIC_SEARCH_PLACEHOLDER = "Search by comic name";

const SERVER = process.env.REACT_APP_PY_SERVER;

const dataFetch = (
    setState, setPagination, from, limit, queryFilter, 
    onlyTracked, onlyUnchecked
  ) => {
  const URL = `${SERVER}/comics/${queryFilter}?from=${from}&limit=${
    limit}&only_tracked=${onlyTracked}&only_unchecked=${onlyUnchecked}`;
  console.debug(URL);
  fetch(URL, {
      method: 'GET',
      headers: { 'accept': 'application/json' },
    })
    .then((response) => {
      console.debug(response)
      setPagination({
        total: response.headers.get('total-comics', 0),
        totalPages: response.headers.get('total-pages', 1),
        currentPage: response.headers.get('current-page', 1)
      });
      return response.json()
    })
    .then((data) => {
      if (data['message'] !== undefined) setState([]);
      else {
        console.debug("Success");
        setState(data);
      }
      console.debug(data);
    })
    .catch((err) => {
      console.log(err.message);
    });
}

export function ComicsMainPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [webComics, setWebComics] = useState([]);
  const [paginationDict, setPaginationDict] = useState({});
    // {from: 0, queryFilter:'', onlyTracked: false, onlyUnchecked: false}
  const onlyUnchecked = searchParams.get('onlyUnchecked') === "true";
  const onlyTracked = searchParams.get('onlyTracked') === "true";
  const queryFilter = searchParams.get('queryFilter') || '';
  const from = parseInt(searchParams.get('from')) || 0;
  const LIMIT = 8;
  
  useEffect(() => {
    dataFetch(
      setWebComics, setPaginationDict,
      from, LIMIT, queryFilter, 
      onlyTracked, onlyUnchecked
    );
  }, [from, queryFilter, onlyTracked, onlyUnchecked]);

  const total = paginationDict.total || 1;
  const totalPages = paginationDict.totalPages || 1;
  const currentPage = paginationDict.currentPage || 1;

  const handleOnlyTracked = () => {
    setSearchParams(prev => {
      prev.set('onlyTracked', !onlyTracked);
      prev.delete('from');
      if (onlyTracked) {
        prev.delete('onlyTracked');
        prev.delete('onlyUnchecked');
      }
      return prev;
    }, {replace: true});
  }

  const handleOnlyUnchecked = () => {
    setSearchParams(prev => {
      prev.set('onlyUnchecked', !onlyUnchecked);
      prev.delete('from');
      if (onlyUnchecked) prev.delete('onlyUnchecked');
      return prev;
    }, {replace: true});
  }

  const handleInputChange = (e) => {
    setSearchParams(prev => {
      if (e?.target?.value !== undefined) 
        prev.set('queryFilter', e?.target?.value);
      return prev;
    }, {replace: true});
  }

  const handlePagination = (direction) => {
    const LAST_FROM = LIMIT*(totalPages-1)
    let moveFrom = 0

    if (direction === 'next') {
      moveFrom = +LIMIT;
    } else if (direction === 'prev') {
      moveFrom = -LIMIT;
    } else if (direction === 'first') {
      moveFrom = -from;
    } else if (direction === 'last') {
      moveFrom = -from + LAST_FROM;
    }  else {
      console.error("Pagination called without valid argument: ", direction);
      return;
    }

    // Border cases, before first page, after last page
    if (from + moveFrom < 0) {
      moveFrom = -from;
    } else if (from + moveFrom > LAST_FROM) {
      moveFrom = -from + LAST_FROM;
    }
    
    setSearchParams(prev => {
      prev.set('from', from + moveFrom);
      if (from + moveFrom === 0) prev.delete('from');
      return prev;
    }, {replace: true});
  }

  return (<>
    <div className='nav-bar'>
      <button className={'basic-button all-track-button' + 
        (onlyTracked ? ' reverse-button' : '')} 
        onClick={handleOnlyTracked} > 
        {onlyTracked ? 'All >' : 'Tracked <'} ({total})
      </button>
      {onlyTracked ?
        (<button className={'basic-button bar-button' + 
          (onlyUnchecked ? ' reverse-button' : '')} 
          onClick={handleOnlyUnchecked} > 
          {onlyUnchecked ? 'No filter' : 'Unchecked'}
        </button>) : ''
      }

      <input className='search-box' type="text" 
        placeholder={COMIC_SEARCH_PLACEHOLDER}
        value={queryFilter} onChange={handleInputChange}
      />

      <div className='div-pagination-buttons'>
        <button className={'basic-button bar-button reverse-button' + 
            (from <= 0 ? ' disabled-button' : '')}
          disabled={from <= 0}
          onClick={() => handlePagination('first')}>
            First
        </button>
        <button className={'basic-button bar-button reverse-button' + 
            (from <= 0 ? ' disabled-button' : '')} 
          disabled={from <= 0}
          onClick={() => handlePagination('prev')}>
            Prev
        </button>
        <button className={'pag-button'}>{currentPage}</button>
        <button className={'basic-button bar-button' +
            (from >= LIMIT*(totalPages-1)
              ? ' disabled-button' : '')}
          disabled={
            from >= LIMIT*(totalPages-1) ? 
              ' disabled-button' : 
              ''
          }
          onClick={() => handlePagination('next')} >
            Next
        </button>
        <button className={'basic-button bar-button' +
            (from >= LIMIT*(totalPages-1)
            ? ' disabled-button' : '')}
          disabled={
            from >= LIMIT*(totalPages-1) ? 
            ' disabled-button' : 
            ''
          }
          onClick={() => handlePagination('last')} >
            Last ({totalPages})
        </button>
      </div>
    </div>

    <ul className='comic-list'> {
      webComics.map( (item, _i) => 
        <ComicCard comic={item} key={item.id} />
      )
    } </ul>
  </>);
};