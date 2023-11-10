import { useState } from 'react';
import styles from'./ComicCard.module.css'
import BrokenImage from '../../../assets/404.jpg'
import { Types, Statuses, Genres, Publishers } from '../../../util/ComicClasses';
import { trackComic, checkoutComic, delComic } from '../../../util/ServerHelpers';
import EditComic from '../Edition/EditComic';

// TODO: Find a solution for image source 
// const CORS_PROXY = 'https://cors-anywhere.herokuapp.com/';

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
  const [comic, setComic] = useState(props.comic);
  const {id, cover, current_chap} = comic;
  const [viewedChap, setViewedChap] = useState(comic.viewed_chap);
  const [track, setTrack] = useState(comic.track);
  const [check, setCheck] = useState(current_chap > viewedChap);
  const [del, setDel] = useState(false);

  const [isHovering, setIsHovering] = useState(false);
  const handleMouseOver = () => setIsHovering(true);;
  const handleMouseOut = () => setIsHovering(false);

  if (del) return;
  return (
    <li key={id} className={styles.comicCard}>
      <div className={styles.posterDiv} 
        style={{backgroundImage: `url(${BrokenImage})`}} >
        <img className={styles.poster} 
          src={cover} 
          alt={comic.titles[0]}
          url={cover}
          onError={(event) => event.currentTarget.src = BrokenImage}
          onMouseOver={handleMouseOver}
          onFocus={handleMouseOver}
          onMouseOut={handleMouseOut}
          onBlur={handleMouseOut}
        />
      </div>

      {isHovering && (<span className={styles.hoverID}>ID: {id}</span>)}

      <h3 className={styles.comicTitle}>{comic.titles[0]}</h3>
      
      <p className={`${styles.comicChapter} text`}> Chapter 
        {track && current_chap !== viewedChap ?
        (<span className={styles.currentChapter}> {viewedChap + '/ '} </span>) 
        : ' '} {current_chap}
      </p>
      
      <p className='text'> {comic.author} </p>
      <p className='text'> Status: {Statuses[comic.status]} </p>
      <p className='text'> Type:   {Types[comic.com_type]} </p>
      <p className='text'> Genres: {genresHandler(comic.genres)} </p>
      <p className='text'> Publishers: {publishersHandler(
        comic.published_in)} </p>

      {track && check ? 
        <button 
          className={`${styles.trackButton} ${styles.checkButton} basic-button`} 
          onClick={() => 
            checkoutComic(current_chap, id, setCheck, setViewedChap)
          }>
          Checkout
        </button> : ''
      }
      <button className={styles.trackButton + ' basic-button' + 
        (track ? ' reverse-button' : '')} 
        onClick={() => trackComic(track, comic.id, setTrack)}>
        {track ? 'Untrack':'Track'}
      </button>
      <button className={styles.delButton + ' basic-button reverse-button'} 
        onClick={() => delComic(comic.id, setDel)}>
        X
      </button>
      <EditComic comic={comic} setComic={setComic} setViewed={setViewedChap} />
    </li>)
};