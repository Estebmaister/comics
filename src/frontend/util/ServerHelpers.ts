import { SetStateAction, Dispatch } from 'react';
import LoadMsgs from '../components/Loaders/LoadMsgs';
import config from './Config';
import { Comic } from '../components/Comics/types';

const SERVER = config.SERVER;

const dataFetch = (
  setters: { 
    setWebComics: Dispatch<SetStateAction<Comic[]>>,
    setPaginationDict: (value: Record<string, any>) => void; 
    setLoadMsg: (value: string | JSX.Element) => void; 
  },
  from: number, limit: number, queryFilter: string,
  onlyTracked: boolean, onlyUnchecked: boolean
) => {
  let BASE_URL = `${SERVER}/comics`;
  queryFilter = queryFilter.trim();
  if (queryFilter !== '') BASE_URL += `/search/${queryFilter}`
  const URL = `${BASE_URL}?from=${from}&limit=${limit}&only_tracked=${onlyTracked}&only_unchecked=${onlyUnchecked}`;
  const { setWebComics, setPaginationDict, setLoadMsg } = setters;
  console.debug(URL);
  setLoadMsg(LoadMsgs.wait);
  fetch(URL, {
    method: 'GET',
    headers: { 'accept': 'application/json' },
  })
    .then((response) => {
      console.debug(response)
      setLoadMsg('');
      setPaginationDict({
        total: response.headers.get('total-comics') || 0,
        totalPages: response.headers.get('total-pages') || 1,
        currentPage: response.headers.get('current-page') || 1
      });
      return response.json()
    })
    .then((data) => {
      if (data['message'] !== undefined) {
        setLoadMsg(LoadMsgs.server);
        setWebComics([]);
      } else setWebComics(data);
      console.debug('Response succeed', data);
    })
    .catch((err) => {
      setLoadMsg(LoadMsgs.network);
      console.log(err.message);
    });
};

const trackComic = (
  tracked: boolean,
  id: number,
  setTrack: { (value: boolean): void; },
  server = SERVER
) => {
  fetch(`${server}/comics/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ track: !tracked }),
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      setTrack(!tracked)
    })
    .catch((err) => {
      console.debug(err.message);
    });
};

const checkoutComic = (
  curr_chap: number,
  id: number,
  setCheck: { (value: SetStateAction<boolean>): void; },
  setViewedChap: { (value: SetStateAction<number>): void; },
  server = SERVER
) => {
  fetch(`${server}/comics/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ viewed_chap: curr_chap }),
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      setCheck(false);
      setViewedChap(curr_chap);
    })
    .catch((err) => {
      console.debug(err.message);
    });
};

const delComic = (
  id: number,
  setDelete: { (value: SetStateAction<boolean>): void; },
  server = SERVER
) => {
  fetch(`${server}/comics/${id}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      setDelete(true);
    })
    .catch((err) => {
      console.debug(err.message);
    });
};

export { dataFetch, trackComic, checkoutComic, delComic };