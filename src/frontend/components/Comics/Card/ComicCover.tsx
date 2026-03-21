import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import BrokenImage from '../../../assets/404.jpg';
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
}

export const ComicCover = ({ comic, children }: ComicCoverProps) => {
  const [imageFailed, setImageFailed] = useState(false);
  const title = comic.titles[0] ?? 'Unknown comic';

  useEffect(() => {
    setImageFailed(false);
  }, [comic.cover]);

  const fallbackBadge = useMemo(() => getCoverBadge(title), [title]);
  const fallbackUsesImage = !title.trim();

  return (
    <div className={styles.posterShell}>
      <div className={styles.posterFrame}>
        {!imageFailed ? (
          <img
            className={styles.posterImage}
            src={comic.cover}
            alt={title}
            loading="lazy"
            decoding="async"
            onError={() => setImageFailed(true)}
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
