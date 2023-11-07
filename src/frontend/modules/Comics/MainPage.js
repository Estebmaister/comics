import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import '../../css/main.css';
import { ComicCard } from './Card/ComicCard';
import CreateComic from './Edition/CreateComic';
import MergeComic from './Edition/MergeComic';
import PaginationButtons from './PaginationButtons';
import { dataFetch, loadMsgs } from '../../util/ServerHelpers';
import ScrapeButton from './Edition/ScrapeButton';

export const COMIC_SEARCH_PLACEHOLDER = 'Search by comic name';

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
  // {from: 0, queryFilter:'', onlyTracked: false, onlyUnchecked: false}
  const [webComics, setWebComics] = useState([]);
  const [paginationDict, setPaginationDict] = useState({});
  const [loadMsg, setLoadMsg] = useState('');
  const onlyUnchecked = searchParams.get('onlyUnchecked') === 'true';
  const onlyTracked = searchParams.get('onlyTracked') === 'true';
  const queryFilter = searchParams.get('queryFilter') || '';
  const from = parseInt(searchParams.get('from')) || 0;
  const limit = 8;
  
  useEffect(() => {
    dataFetch(
      {setWebComics, setPaginationDict, setLoadMsg},
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
      <h1 className='server'> {loadMsg || loadMsgs.empty(queryFilter)} </h1>
    }
    <ul className='comic-list'> {
      webComics.map((item, _i) => <ComicCard comic={item} key={item.id} />)
    } </ul>

    <MergeComic />
    <CreateComic />
    <ScrapeButton />
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