import React from 'react';
import { useSearchParams } from 'react-router-dom';
import '../css/main.css';
import comics from '../../db/comics.json';
import { ComicCard } from './ComicCard';

comics.sort((a,b) => b.last_update - a.last_update)
export const COMIC_SEARCH_PLACEHOLDER = "Search by comic name";

const filterComics = (comics, filterWord, trackedOnly, uncheckedOnly) => 
  comics.filter((comic) => {
    for (const title of comic.titles) {
      if (trackedOnly && !comic.track) {
        return false;
      }
      if (uncheckedOnly && (comic.viewed_chap === comic.current_chap)) {
        return false;
      }
      if (title.toLowerCase().includes( filterWord.toLowerCase() )) {
        return true;
      }
    }
    return false;
  }
);

export function ComicsMainPage() {
  const [searchParams, setSearchParams] = useSearchParams(
    //{queryFilter:'', onlyTracked: false, from: 0, onlyUnchecked: false}
  );
  const queryFilter = searchParams.get('queryFilter') || '';
  const onlyTracked = searchParams.get('onlyTracked') === "true";
  const onlyUnchecked = searchParams.get('onlyUnchecked') === "true";
  const from = parseInt(searchParams.get('from')) || 0;
  const LIMIT = 8;
  
  const FILTERED_COMICS = filterComics(comics, queryFilter, onlyTracked);
  let total = FILTERED_COMICS.length;
  
  const totalPages = Math.ceil(total/LIMIT);
  const currentPage = Math.ceil(from/LIMIT +1)

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

  let filteredComics;
  if (queryFilter !== '' || onlyTracked === true) {
    filteredComics = filterComics(comics, queryFilter, onlyTracked, onlyUnchecked);
    total = filteredComics.length;
    filteredComics = filteredComics.slice(from, from + LIMIT);
  } else {
    total = comics.length;
    filteredComics = comics.slice(from, from + LIMIT);
  };

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
      filteredComics.map( (item, _i) => 
        <ComicCard comic={item} key={item.id} />
      )
    } </ul>
  </>);
};