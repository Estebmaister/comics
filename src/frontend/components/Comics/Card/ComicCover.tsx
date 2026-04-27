import { useEffect, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import BrokenImage from '../../../assets/404.jpg';
import { reportCoverVisibility } from '../../../util/ServerHelpers';
import type { Comic } from '../types';
import styles from './ComicCard.module.css';

const getCoverBadge = (title: string) => title
  .split(/\s+/)
  .filter(Boolean)
  .slice(0, 2)
  .map((chunk) => chunk[0]?.toUpperCase() ?? '')
  .join('') || '404';

interface ComicCoverProps {
  comic: Comic;
  children: ReactNode;
  onCoverVisibilityChange?: (coverVisible: boolean) => void;
}

export const ComicCover = ({
  comic,
  children,
  onCoverVisibilityChange,
}: ComicCoverProps) => {
  const [imageFailed, setImageFailed] = useState(false);
  const reportedFailuresRef = useRef<Set<string>>(new Set());
  const title = comic.titles[0] ?? 'Unknown comic';

  useEffect(() => {
    setImageFailed(false);
  }, [comic.cover, comic.cover_visible]);

  const handleCoverError = () => {
    setImageFailed(true);
    const failedCover = comic.cover;
    const failureKey = `${comic.id}:${failedCover}`;
    if (!failedCover || reportedFailuresRef.current.has(failureKey)) return;

    reportedFailuresRef.current.add(failureKey);
    reportCoverVisibility(comic.id, failedCover, false)
      .then((updatedComic) => {
        if (updatedComic?.cover === failedCover) {
          onCoverVisibilityChange?.(updatedComic.cover_visible !== false);
        }
      })
      .catch((error) => {
        console.debug(error);
      });
  };

  const fallbackBadge = useMemo(() => getCoverBadge(title), [title]);
  const fallbackUsesImage = !title.trim();
  const shouldLoadCover = comic.cover_visible !== false && !imageFailed && Boolean(comic.cover);

  return (
    <div className={styles.posterShell}>
      <div className={styles.posterFrame}>
        {shouldLoadCover ? (
          <img
            className={styles.posterImage}
            src={comic.cover}
            alt={title}
            loading="lazy"
            decoding="async"
            onError={handleCoverError}
          />
        ) : fallbackUsesImage ? (
          <img
            className={styles.posterImage}
            src={BrokenImage}
            alt="Fallback comic cover"
            loading="lazy"
            decoding="async"
          />
        ) : (
          <div className={styles.posterFallback} data-testid="poster-fallback">
            <span className={styles.posterFallbackBadge}>{fallbackBadge}</span>
            <strong className={styles.posterFallbackTitle}>{title}</strong>
            <span className={styles.posterFallbackMeta}>Cover unavailable</span>
          </div>
        )}

        <div className={styles.posterOverlay}>
          {children}
        </div>
      </div>
    </div>
  );
};
