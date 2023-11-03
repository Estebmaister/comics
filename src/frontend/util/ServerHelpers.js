import Loaders from '../modules/Loaders';
const SERVER = process.env.REACT_APP_PY_SERVER;

const dataFetch = (
    setters, from, limit, queryFilter, 
    onlyTracked, onlyUnchecked
  ) => {
  let BASE_URL = `${SERVER}/comics`;
  queryFilter = queryFilter.trim();
  if (queryFilter !== '') BASE_URL += `/search/${queryFilter}`
  const URL = `${BASE_URL}?from=${from}&limit=${
    limit}&only_tracked=${onlyTracked}&only_unchecked=${onlyUnchecked}`;
  const {setWebComics, setPaginationDict, setLoadMsg} = setters;
  console.debug(URL);
  setLoadMsg(loadMsgs.wait);
  fetch(URL, {
      method: 'GET',
      headers: { 'accept': 'application/json' },
    })
    .then((response) => {
      console.debug(response)
      setLoadMsg('');
      setPaginationDict({
        total: response.headers.get('total-comics', 0),
        totalPages: response.headers.get('total-pages', 1),
        currentPage: response.headers.get('current-page', 1)
      });
      return response.json()
    })
    .then((data) => {
      if (data['message'] !== undefined) {
        setLoadMsg(loadMsgs.server);
        setWebComics([]);
      } else setWebComics(data);
      console.debug('Response succeed', data);
    })
    .catch((err) => {
      setLoadMsg(loadMsgs.network);
      console.log(err.message);
    });
};

const trackComic = (tracked, id, setTrack, server = SERVER) => {
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

const checkoutComic = (curr_chap, id, setCheck, setViewedChap, server = SERVER) => {
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

const delComic = (id, setDelete, server = SERVER) => {
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

const loadMsgs = {
  network: <>
    {'Network error in attempt to connect the server'} 
    <Loaders selector='lamp' />
  </>,
  server: 'Server internal error',
  wait: <>{'Waking up server ...'} <Loaders selector='line-fw' /></>,
  empty: (queryFilter) => `No comics found for title: ${queryFilter}`
}

export { dataFetch, trackComic, checkoutComic, delComic, loadMsgs };