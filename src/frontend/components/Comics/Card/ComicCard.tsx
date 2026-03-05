import { JSX, memo, useMemo, useState } from 'react';
import styles from './ComicCard.module.css'
import BrokenImage from '../../../assets/404.jpg'
import { Types, Statuses } from '../../../util/ComicClasses';
import { genresHandler, publishersHandler } from './ComicFormatters';
import EditComic from '../Edition/EditComic';
import CopyableSpan from './CopyableSpan';
import { useComicActions } from '../../../hooks/useComicActions';
import { ComicCardProvider } from './ComicCardContext';

// TODO: Research a solution for image sourcing
// const CORS_PROXY = 'https://cors-anywhere.herokuapp.com/';

interface ComicCardProps {
  comic: Record<string, any>;
  onCheckoutSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

const ComicCard = (props: ComicCardProps): JSX.Element | null => {
  const [comic, setComic] = useState(props.comic);
  const { id, cover, current_chap } = comic;
  const [viewedChap, setViewedChap] = useState<number>(comic.viewed_chap);
  const [check, setCheck] = useState(current_chap > viewedChap);
  const [del, setDel] = useState(false);
  const showCheckout = useMemo(() => comic.track && check, [comic.track, check]);

  const { handleCheckout, handleTrackToggle, handleDelete } = useComicActions({
    comicId: id,
    currentChap: current_chap,
    isTracked: comic.track,
    setComic,
    setViewedChap,
    setCheck,
    setDel,
    onCheckoutSuccess: props.onCheckoutSuccess,
    onDeleteSuccess: props.onDeleteSuccess,
  });

  const [isHovering, setIsHovering] = useState(false);
  const handleMouseOver = () => setIsHovering(true);;
  const handleMouseOut = () => setIsHovering(false);

  if (del) return null;
  return (
    <ComicCardProvider comic={comic} setComic={setComic} setViewedChap={setViewedChap}>
      <li key={id} className={styles.comicCard}>
        <div className={styles.cardRow}>
          <div
            className={styles.posterDiv}
            style={{ backgroundImage: `url(${BrokenImage})` }}
          >
            <img className={styles.poster}
              src={cover}
              alt={comic.titles[0]}
              datatype={cover}
              loading="lazy"
              decoding="async"
              onError={(event) => event.currentTarget.src = BrokenImage}
              onMouseOver={handleMouseOver}
              onFocus={handleMouseOver}
              onMouseOut={handleMouseOut}
              onBlur={handleMouseOut}
            />

            <button
              className={styles.delButton}
              onClick={handleDelete}
            >
              X
            </button>
            <EditComic />

            {isHovering && (
              <CopyableSpan
                textToCopy={id}
                textToShow={`ID: ${id}`}
                className={`${styles.hoverID} basic-button`}
                onMouseOver={handleMouseOver}
                onFocus={handleMouseOver}
                onMouseOut={handleMouseOut}
                onBlur={handleMouseOut}
              />
            )}
          </div>

          <div className={styles.contentColumn}>
            <h3 className={styles.comicTitle}>{comic.titles[0]}</h3>

            <p className={styles.comicChapter}>
              <span className={styles.fieldLabel}>Chapter</span>{' '}
              {comic.track && current_chap !== viewedChap ?
                (<span className={styles.currentChapter}>{viewedChap}/ </span>)
                : null}
              <span className={styles.chapterValue}>{current_chap}</span>
            </p>

            <p className={styles.metaLine}>{comic.author}</p>
            <p className={styles.metaLine}>
              <span className={styles.fieldLabel}>Status:</span> {Statuses[comic.status]}
            </p>
            <p className={styles.metaLine}>
              <span className={styles.fieldLabel}>Type:</span> {Types[comic.com_type]}
            </p>
            <p className={styles.metaLine}>
              <span className={styles.fieldLabel}>Genres:</span> {genresHandler(comic.genres)}
            </p>

            <div className={styles.footerRow}>
              <p className={styles.publisherText}>
                <span className={styles.fieldLabel}>Publishers:</span> {publishersHandler(comic.published_in)}
              </p>
              <div className={styles.actionsColumn}>
                {showCheckout ? (
                  <button
                    className={`${styles.actionButton} ${styles.checkButton} basic-button`}
                    onClick={handleCheckout}
                  >
                    Checkout
                  </button>
                ) : null}
                <button
                  className={`${styles.actionButton} basic-button${comic.track ? ' reverse-button' : ''}`}
                  onClick={handleTrackToggle}
                >
                  {comic.track ? 'Untrack' : 'Track'}
                </button>
              </div>
            </div>
          </div>
        </div>
      </li>
    </ComicCardProvider>
  );
};

export default memo(ComicCard);
