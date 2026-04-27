import { JSX, memo, useMemo, useState } from 'react';
import styles from './ComicCard.module.css';
import { Types, Statuses } from '../../../util/ComicClasses';
import { genresHandler, publishersHandler } from './ComicFormatters';
import EditComic from '../Edition/EditComic';
import CopyableSpan from './CopyableSpan';
import { useComicActions } from '../../../hooks/useComicActions';
import { ComicCardProvider } from './ComicCardContext';
import { ComicCover } from './ComicCover';
import type { Comic } from '../types';

interface ComicCardProps {
  comic: Comic;
  onCheckoutSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

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
              <h3 className={styles.comicTitle}>{title}</h3>
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

export default memo(ComicCard);
