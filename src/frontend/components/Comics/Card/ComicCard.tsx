import { JSX, memo, useEffect, useMemo, useState } from 'react';
import styles from './ComicCard.module.css';
import { Types, Statuses } from '../../../util/ComicClasses';
import { genresHandler, publishersHandler } from './ComicFormatters';
import EditComic from '../Edition/EditComic';
import CopyableSpan from './CopyableSpan';
import { useComicActions } from '../../../hooks/useComicActions';
import { ComicCardProvider } from './ComicCardContext';
import { ComicCover } from './ComicCover';
import { rateComic } from '../../../util/ServerHelpers';
import type { Comic } from '../types';

interface ComicCardProps {
  comic: Comic;
  onCheckoutSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

const arraysEqual = (left: readonly unknown[] = [], right: readonly unknown[] = []) => (
  left.length === right.length && left.every((value, index) => value === right[index])
);

const comicsEqual = (left: Comic, right: Comic) => (
  left.id === right.id
  && arraysEqual(left.titles, right.titles)
  && left.cover === right.cover
  && left.cover_visible === right.cover_visible
  && left.author === right.author
  && left.current_chap === right.current_chap
  && left.viewed_chap === right.viewed_chap
  && left.track === right.track
  && left.status === right.status
  && left.com_type === right.com_type
  && arraysEqual(left.genres, right.genres)
  && arraysEqual(left.published_in, right.published_in)
  && left.description === right.description
  && left.rating === right.rating
  && left.deleted === right.deleted
  && left.last_update === right.last_update
);

const comicCardPropsEqual = (left: ComicCardProps, right: ComicCardProps) => (
  comicsEqual(left.comic, right.comic)
  && left.onCheckoutSuccess === right.onCheckoutSuccess
  && left.onDeleteSuccess === right.onDeleteSuccess
);

const ComicCard = ({
  comic: initialComic,
  onCheckoutSuccess,
  onDeleteSuccess,
}: ComicCardProps): JSX.Element | null => {
  const [comic, setComic] = useState(initialComic);
  const { id, current_chap } = comic;
  const [viewedChap, setViewedChap] = useState<number>(comic.viewed_chap);
  const [check, setCheck] = useState(current_chap > viewedChap);
  const [del, setDel] = useState(false);
  const showCheckout = useMemo(() => comic.track && check, [comic.track, check]);
  const title = comic.titles[0] ?? 'Unknown comic';
  const rating = Math.max(0, Math.min(5, Number(comic.rating ?? 0)));

  const handleRate = (next: number) => {
    if (Number(comic.rating ?? 0) === next) return;
    void rateComic(next, id, setComic);
  };

  useEffect(() => {
    setComic(initialComic);
    setViewedChap(initialComic.viewed_chap);
    setCheck(initialComic.current_chap > initialComic.viewed_chap);
    setDel(false);
  }, [initialComic]);

  const { handleCheckout, handleTrackToggle, handleDelete } = useComicActions({
    comicId: id,
    currentChap: current_chap,
    isTracked: comic.track,
    setComic,
    setViewedChap,
    setCheck,
    setDel,
    onCheckoutSuccess,
    onDeleteSuccess,
  });

  const genreText = useMemo(() => genresHandler(comic.genres), [comic.genres]);
  const publisherLinks = useMemo(
    () => publishersHandler(comic.published_in),
    [comic.published_in]
  );

  if (del) return null;

  return (
    <ComicCardProvider comic={comic} setComic={setComic} setViewedChap={setViewedChap}>
      <li className={styles.comicCard}>
        <div className={styles.cardGlow} />

        <div className={styles.cardRow}>
          <ComicCover
            comic={comic}
            onCoverVisibilityChange={(cover_visible) => {
              setComic((prev) => ({ ...prev, cover_visible }));
            }}
          >
            <div className={styles.overlayActions}>
              <button
                className={`${styles.overlayButton} ${styles.overlayButtonDanger}`}
                onClick={handleDelete}
                aria-label={`Delete ${title}`}
              >
                Delete
              </button>
              <EditComic className={styles.overlayButton}>Edit</EditComic>
            </div>

            <CopyableSpan
              textToCopy={id}
              textToShow={`ID ${id}`}
              className={styles.idChip}
              ariaLabel={`Copy comic ID ${id}`}
            />
          </ComicCover>

          <div className={styles.contentColumn}>
            <div className={styles.headingBlock}>
              <div className={styles.titleRow}>
                <h3 className={styles.comicTitle}>{title}</h3>
                <div
                  className={styles.ratingChip}
                  role="group"
                  aria-label={`Rating for ${title}`}
                >
                  <span className={styles.ratingDots}>
                    {Array.from({ length: 5 }, (_, index) => {
                      const value = index + 1;
                      const active = value <= rating;
                      return (
                        <button
                          key={value}
                          type="button"
                          className={styles.ratingPip}
                          onClick={() => handleRate(rating === value ? 0 : value)}
                          aria-label={`Set rating to ${value} of 5`}
                          aria-pressed={active}
                          title={`${value}/5`}
                        >
                          <span
                            className={`${styles.ratingDot}${active ? ` ${styles.ratingDotActive}` : ''}`}
                          />
                        </button>
                      );
                    })}
                  </span>
                </div>
              </div>
              <p className={styles.authorLine}>
                {comic.author || 'Author unknown'}
              </p>
            </div>

            <p className={styles.comicChapter}>
              <span className={styles.fieldLabel}>Chapter</span>
              {comic.track && current_chap !== viewedChap ? (
                <span className={styles.chapterProgress}>{viewedChap}/{current_chap}</span>
              ) : (
                <span className={styles.chapterValue}>{current_chap}</span>
              )}
            </p>

            <dl className={styles.metaGrid}>
              <div className={styles.metaItem}>
                <dt className={styles.metaLabel}>Status</dt>
                <dd className={styles.metaValue}>{Statuses[comic.status]}</dd>
              </div>
              <div className={styles.metaItem}>
                <dt className={styles.metaLabel}>Type</dt>
                <dd className={styles.metaValue}>{Types[comic.com_type]}</dd>
              </div>
              <div className={`${styles.metaItem} ${styles.metaItemWide}`}>
                <dt className={styles.metaLabel}>Genres</dt>
                <dd className={`${styles.metaValue} ${styles.metaClamp}`}>{genreText}</dd>
              </div>
            </dl>

            <div className={styles.footerRow}>
              <p className={styles.publisherText}>
                <span className={styles.fieldLabel}>Publishers</span>
                <span className={styles.publisherLinks}>{publisherLinks}</span>
              </p>

              <div className={styles.actionsColumn} data-testid="comic-footer-actions">
                {showCheckout ? (
                  <button
                    className={`${styles.actionButton} ${styles.checkoutButton} basic-button`}
                    onClick={handleCheckout}
                  >
                    Checkout
                  </button>
                ) : (
                  <span className={styles.actionButtonPlaceholder} aria-hidden="true" />
                )}
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

export default memo(ComicCard, comicCardPropsEqual);
