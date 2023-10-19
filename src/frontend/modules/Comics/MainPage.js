import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../../css/main.css';
import { ComicCard } from './Card/ComicCard';
import CreateComic from './Create/CreateComic';

export const COMIC_SEARCH_PLACEHOLDER = 'Search by comic name';

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
        console.debug('Success');
        setState(data);
      }
      console.debug(data);
    })
    .catch((err) => {
      console.log(err.message);
    });
}

const handleOnlyUnchecked = (setSearchParams, onlyUnchecked) => 
  () => {
    setSearchParams(prev => {
      prev.set('onlyUnchecked', !onlyUnchecked);
      prev.delete('from');
      if (onlyUnchecked) prev.delete('onlyUnchecked');
      return prev;
    }, {replace: true});
  };

const handleOnlyTracked = (setSearchParams, onlyTracked) => 
  () => {
    setSearchParams(prev => {
      prev.set('onlyTracked', !onlyTracked);
      prev.delete('from');
      if (onlyTracked) {
        prev.delete('onlyTracked');
        prev.delete('onlyUnchecked');
      }
      return prev;
    }, {replace: true});
  };

export function ComicsMainPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [webComics, setWebComics] = useState([]);
  const [paginationDict, setPaginationDict] = useState({});
    // {from: 0, queryFilter:'', onlyTracked: false, onlyUnchecked: false}
  const onlyUnchecked = searchParams.get('onlyUnchecked') === 'true';
  const onlyTracked = searchParams.get('onlyTracked') === 'true';
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

  const onFirstPage = from <= 0;
  const onLastPage = from >= LIMIT*(totalPages-1);

  const handleInputChange = (e) => {
    setSearchParams(prev => {
      if (e?.target?.value !== undefined) 
        prev.set('queryFilter', e?.target?.value);
      prev.set('from', 0);
      return prev;
    }, {replace: true});
  }

  const handlePagination = (direction) => {
    const LAST_FROM = LIMIT*(totalPages-1)
    let moveFrom = 0

    if      (direction === 'next' ) moveFrom = +LIMIT;
    else if (direction === 'prev' ) moveFrom = -LIMIT;
    else if (direction === 'first') moveFrom = -from;
    else if (direction === 'last' ) moveFrom = -from + LAST_FROM;
    else {
      console.error('Pagination called without valid argument: ', direction);
      return;
    }

    // Border cases, before first page, after last page
    if      (from + moveFrom < 0        ) moveFrom = -from;
    else if (from + moveFrom > LAST_FROM) moveFrom = -from + LAST_FROM;
    
    setSearchParams(prev => {
      prev.set('from', from + moveFrom);
      if (from + moveFrom === 0) prev.delete('from');
      return prev;
    }, {replace: true});
  }

  return (<>
    <div className='nav-bar'>
      <ConditionalButton 
        condFlag={onlyTracked} extraClass={'reverse-button'}
        onClick={handleOnlyTracked(setSearchParams, onlyTracked)}
        positiveMsg={`All > (${total})`} negativeMsg={`Tracked < (${total})`} 
        className={'basic-button all-track-button'}
      />
      <ConditionalButton 
        showFlag={onlyTracked} condFlag={onlyUnchecked}
        onClick={handleOnlyUnchecked(setSearchParams, onlyUnchecked)}
        positiveMsg={'No filter'} negativeMsg={'Unchecked'} 
        className={'basic-button bar-button'} extraClass={'reverse-button'}
      />

      <input className='search-box' placeholder={COMIC_SEARCH_PLACEHOLDER}
        type='text' value={queryFilter} onChange={handleInputChange}
      />

      <div className='div-pagination-buttons'>
        <button className={'basic-button bar-button reverse-button' + 
            (onFirstPage ? ' disabled-button' : '')}
          disabled={onFirstPage} onClick={() => handlePagination('first')}>
            First
        </button>
        <button className={'basic-button bar-button reverse-button' + 
            (onFirstPage ? ' disabled-button' : '')} 
          disabled={onFirstPage} onClick={() => handlePagination('prev')}>
            Prev
        </button>
        <button className={'pag-button'}> {currentPage} </button>
        <button className={'basic-button bar-button' +
            (onLastPage ? ' disabled-button' : '')}
          disabled={onLastPage} onClick={() => handlePagination('next')} >
            Next
        </button>
        <button className={'basic-button bar-button' +
            (onLastPage ? ' disabled-button' : '')}
          disabled={onLastPage} onClick={() => handlePagination('last')} >
            Last ({totalPages})
        </button>
      </div>
    </div>

    <ul className='comic-list'> {
      webComics.map((item, _i) => <ComicCard comic={item} key={item.id} />)
    } </ul>

    <CreateComic />
  </>);
};

const ConditionalButton = ({ showFlag=true, onClick, condFlag, disabled,
  positiveMsg, negativeMsg = positiveMsg, className, extraClass  }) => {

  return <> { showFlag ?
    <button className={`${className}` + (condFlag ? ` ${extraClass}` : '')} 
      onClick={onClick} disabled={disabled}> 
      {condFlag ? `${positiveMsg}` : `${negativeMsg}`}
    </button> : ''
  } </>
};