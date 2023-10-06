import { Types, Statuses, Genres, Publishers } from '../util/ComicClasses';
import { useState } from 'react';
import BrokenImage from '../assets/404.jpg'
import styles from'./ComicCard.module.css'
const SERVER = process.env.REACT_APP_PY_SERVER;
// TODO: const CORS_PROXY = 'https://cors-anywhere.herokuapp.com/';

const trackFunc = (tracked, id, setTrack, server = SERVER) => {
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
}

const checkout = (curr_chap, id, setCheck, setViewedChap, server = SERVER) => {
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
}

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
}

const publishersHandler = (publishers) => {
    const return_array = [];
    publishers.forEach( element => 
        return_array.push(Publishers[+element]) )
    return return_array.join(', ');
}

const genresHandler = (genres) => {
    const return_array = [];
    genres.forEach( element => 
        return_array.push(Genres[+element]) )
    return return_array.join(', ');
}

export const ComicCard = (props) => {
  const {id, cover, current_chap} = props.comic;
  const [viewedChap, setViewedChap] = useState(props.comic.viewed_chap);
  const [track, setTrack] = useState(props.comic.track);
  const [check, setCheck] = useState(current_chap > viewedChap);
  const [del, setDel] = useState(false);
  if (del) return;
  return (
    <li key={id} className={styles.comicCard}>
      <img className={styles.poster} 
        src={cover} 
        alt={props.comic.titles[0]}
        url={cover}
        onError={(event) => event.currentTarget.src = BrokenImage} 
      />
      {/* TODO: Show ID on hover */}
      <h3 className={styles.comicTitle}>{props.comic.titles[0]}</h3>
      
      <p className={`${styles.comicChapter} text`}> Chapter 
        {track && current_chap !== viewedChap ?
        (<span className={styles.currentChapter}> {viewedChap + '/ '} </span>) 
        : ' '} {current_chap}
      </p>
      
      <p className='text'> {props.comic.author} </p>
      <p className='text'> Status: {Statuses[props.comic.status]} </p>
      <p className='text'> Type:   {Types[props.comic.com_type]} </p>
      <p className='text'> Genres: {genresHandler(props.comic.genres)} </p>
      <p className='text'> Publishers: {publishersHandler(
        props.comic.published_in)} </p>

      {track && check ? 
        <button 
          className={`${styles.trackButton} ${styles.checkButton} basic-button`} 
          onClick={() => checkout(current_chap, id, setCheck, setViewedChap)}>
          Checkout
        </button> : ''
      }
      <button className={styles.trackButton + ' basic-button' + 
        (track ? ' reverse-button' : '')} 
        onClick={() => trackFunc(track, props.comic.id, setTrack)}>
        {track ? 'Untrack':'Track'}
      </button>
      <button className={styles.delButton + ' basic-button reverse-button'} 
        onClick={() => delComic(props.comic.id, setDel)}>
        X
      </button>
    </li>)
};