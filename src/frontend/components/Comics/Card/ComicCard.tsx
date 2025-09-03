import { JSX, useState } from 'react';
import styles from './ComicCard.module.css'
import BrokenImage from '../../../assets/404.jpg'
import { Types, Statuses } from '../../../util/ComicClasses';
import { trackComic, checkoutComic, delComic } from '../../../util/ServerHelpers';
import { genresHandler, publishersHandler } from './ComicFormatters';
import EditComic from '../Edition/EditComic';
import CopyableSpan from './CopyableSpan';

// TODO: Research a solution for image sourcing
// const CORS_PROXY = 'https://cors-anywhere.herokuapp.com/';

const ComicCard = (props: { comic: Record<string, any>; }): JSX.Element | null => {
  const [comic, setComic] = useState(props.comic);
  const { id, cover, current_chap } = comic;
  const [viewedChap, setViewedChap] = useState(comic.viewed_chap);
  const [check, setCheck] = useState(current_chap > viewedChap);
  const [del, setDel] = useState(false);

  const setTrackComic = (track: boolean) => {
    setComic({ ...comic, track: track });
  }

  const [isHovering, setIsHovering] = useState(false);
  const handleMouseOver = () => setIsHovering(true);;
  const handleMouseOut = () => setIsHovering(false);

  if (del) return null;
  return (<li key={id} className={styles.comicCard}>
    <div className={styles.posterDiv}
      style={{ backgroundImage: `url(${BrokenImage})` }} >
      <img className={styles.poster}
        src={cover}
        alt={comic.titles[0]}
        datatype={cover}
        onError={(event) => event.currentTarget.src = BrokenImage}
        onMouseOver={handleMouseOver}
        onFocus={handleMouseOver}
        onMouseOut={handleMouseOut}
        onBlur={handleMouseOut}
      />
    </div>

    {isHovering && (
      <CopyableSpan
        textToCopy={id}
        textToShow={`ID: ${id}`}
        className={styles.hoverID}
        onMouseOver={handleMouseOver}
        onFocus={handleMouseOver}
        onMouseOut={handleMouseOut}
        onBlur={handleMouseOut}
      />)
    }

    <h3 className={styles.comicTitle}>{comic.titles[0]}</h3>

    <p className={`${styles.comicChapter} text`}> Chapter
      {comic.track && current_chap !== viewedChap ?
        (<span className={styles.currentChapter}> {viewedChap + '/ '} </span>)
        : ' '} {current_chap}
    </p>

    <p className='text'> {comic.author} </p>
    <p className='text'> Status: {Statuses[comic.status]} </p>
    <p className='text'> Type:   {Types[comic.com_type]} </p>
    <p className='text'> Genres: {genresHandler(comic.genres)} </p>
    <p className='text'> Publishers: {publishersHandler(comic.published_in)} </p>

    {comic.track && check ?
      <button
        className={`${styles.trackButton} ${styles.checkButton} basic-button`}
        onClick={() => checkoutComic(current_chap, id, setCheck, setViewedChap)}>
        Checkout
      </button> : ''
    }
    <button
      className={styles.trackButton + ' basic-button' + (comic.track ? ' reverse-button' : '')}
      onClick={() => trackComic(comic.track, comic.id, setTrackComic)}>
      {comic.track ? 'Untrack' : 'Track'}
    </button>
    <button
      className={styles.delButton + ' basic-button reverse-button'}
      onClick={() => delComic(comic.id, setDel)}>
      X
    </button>
    <EditComic comic={comic} setComic={setComic} setViewed={setViewedChap} />
  </li>)
};

export default ComicCard;