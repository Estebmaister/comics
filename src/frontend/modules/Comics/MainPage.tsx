import { ChangeEvent, useEffect, useState } from 'react';
import { SetURLSearchParams, useSearchParams } from 'react-router-dom';
import '../../css/main.css';
import { ComicCard } from './Card/ComicCard';
import CreateComic from './Edition/CreateComic';
import MergeComic from './Edition/MergeComic';
import PaginationButtons from './PaginationButtons';
import { dataFetch, loadMsgs } from '../../util/ServerHelpers';
import ScrapeButton from './Edition/ScrapeButton';

export const COMIC_SEARCH_PLACEHOLDER = 'Search by comic name';

const handleOnlyUnchecked = (
  setSearchParams: SetURLSearchParams, onlyUnchecked: boolean
) => () => {
  setSearchParams(prev => {
    prev.set('onlyUnchecked', String(!onlyUnchecked));
    prev.delete('from');
    if (onlyUnchecked) prev.delete('onlyUnchecked');
    return prev;
  }, { replace: false });
};

const handleOnlyTracked = (
  setSearchParams: SetURLSearchParams,
  onlyTracked: boolean
) => () => {
  setSearchParams(prev => {
    prev.set('onlyTracked', String(!onlyTracked));
    prev.delete('from');
    if (onlyTracked) {
      prev.delete('onlyTracked');
      prev.delete('onlyUnchecked');
    }
    return prev;
  }, { replace: false });
};

// https://www.dhiwise.com/post/react-get-screen-width-everything-you-need-to-know
const width = window.innerWidth;
const comicCardWidth = 450;
const mainPagePadding = 30;
const inlineComicsWOP = Math.floor(width / comicCardWidth);
const inlineComics = width - (comicCardWidth * inlineComicsWOP + mainPagePadding) >= 0 ?
  inlineComicsWOP : inlineComicsWOP - 1;

type mainState = { [key: string]: number };

export function ComicsMainPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  // {from: 0, queryFilter:'', onlyTracked: false, onlyUnchecked: false}
  const [webComics, setWebComics] = useState([]);
  const [paginationDict, setPaginationDict] = useState<mainState>({});
  const [loadMsg, setLoadMsg] = useState('');
  const onlyUnchecked = searchParams.get('onlyUnchecked') === 'true';
  const onlyTracked = searchParams.get('onlyTracked') === 'true';
  const queryFilter = searchParams.get('queryFilter') || '';
  const from = parseInt(searchParams.get('from') || '0') || 0;
  const limit = inlineComics * 3;

  useEffect(() => {
    dataFetch(
      { setWebComics, setPaginationDict, setLoadMsg },
      from, limit, queryFilter,
      onlyTracked, onlyUnchecked
    );
  }, [from, limit, queryFilter, onlyTracked, onlyUnchecked]);

  const total = paginationDict.total || 1;
  const totalPages = paginationDict.totalPages || 1;
  const currentPage = paginationDict.currentPage || 1;

  const onFirstPage = from <= 0;
  const onLastPage = from >= limit * (totalPages - 1);

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    setSearchParams(prev => {
      if (e?.target?.value !== undefined)
        prev.set('queryFilter', e?.target?.value);
      prev.set('from', '0');
      return prev;
    }, { replace: true });
  };

  const pagD = {
    from, limit, setSearchParams,
    onFirstPage, onLastPage, currentPage, totalPages
  };

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

      <PaginationButtons pagD={pagD} />
    </div>

    {webComics.length === 0 &&
      <h1 className='server'> {loadMsg || loadMsgs.empty(queryFilter)} </h1>
    }
    <ul className='comic-list'> {
      webComics.map((comic: any, _i) => <ComicCard comic={comic} key={comic.id} />)
    } </ul>

    <MergeComic />
    <CreateComic />
    <ScrapeButton />
  </>);
};


const ConditionalButton = ({ showFlag = true, onClick, condFlag = false, disabled = false,
  positiveMsg = '', negativeMsg = positiveMsg, className = '', extraClass = '' }: any) => {

  return <> {showFlag ?
    <button className={`${className}` + (condFlag ? ` ${extraClass}` : '')}
      onClick={onClick} disabled={disabled}>
      {condFlag ? `${positiveMsg}` : `${negativeMsg}`}
    </button> : ''
  } </>
};