import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../../css/main.css';
import { ComicCard } from './Card/ComicCard';
import CreateComic from './Create/CreateComic';
import PaginationButtons from './PaginationButtons';

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
      else setState(data);
      console.debug('Response succeed', data);
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
  const limit = 8;
  
  useEffect(() => {
    dataFetch(
      setWebComics, setPaginationDict,
      from, limit, queryFilter, 
      onlyTracked, onlyUnchecked
    );
  }, [from, queryFilter, onlyTracked, onlyUnchecked]);

  const total = paginationDict.total || 1;
  const totalPages = paginationDict.totalPages || 1;
  const currentPage = paginationDict.currentPage || 1;

  const onFirstPage = from <= 0;
  const onLastPage = from >= limit*(totalPages-1);

  const handleInputChange = (e) => {
    setSearchParams(prev => {
      if (e?.target?.value !== undefined) 
        prev.set('queryFilter', e?.target?.value);
      prev.set('from', 0);
      return prev;
    }, {replace: true});
  };

  const pagD = { from, limit, setSearchParams,
    onFirstPage, onLastPage, currentPage, totalPages };

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

      <PaginationButtons pagD={pagD}/>
    </div>

    { webComics.length === 0 && 
      <h1 className='server'> {queryFilter === '' ? 
        'Waking up server ...' : `No comics found for title: ${queryFilter}`}
      </h1>
    }
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