import { FormEvent, useEffect, useMemo, useState } from 'react';
import Modal from '../../Modal';
import '../Modals/Modal.css';
import type { Comic } from '../types';
import type { CatalogConflictPayload } from '../../../util/catalogConflict';
import {
  conflictHeadline,
  matchKindLabel,
  mergePairFromConflict,
} from '../../../util/catalogConflict';
import { fetchComicById } from '../../../util/comicUpdate';
import { mergeComic } from '../Edition/MergeComic';

type DiscoveryConflictModalProps = {
  isOpen: boolean;
  conflict: CatalogConflictPayload | null;
  onClose: () => void;
  onDismiss?: () => void;
  onMerged: (removedComicId: number | null) => void;
  onRetry?: () => Promise<boolean>;
};

type ComicPreview = {
  id: number;
  title: string;
  chapter: number;
  comType: number;
  track: boolean;
  rating: number;
};

const toPreview = (comic: Comic): ComicPreview => ({
  id: comic.id,
  title: comic.titles[0] ?? `Comic ${comic.id}`,
  chapter: comic.current_chap,
  comType: comic.com_type,
  track: comic.track,
  rating: Math.max(0, Math.min(5, Number(comic.rating ?? 0))),
});

const DiscoveryConflictModal = ({
  isOpen,
  conflict,
  onClose,
  onDismiss,
  onMerged,
  onRetry,
}: DiscoveryConflictModalProps) => {
  const [previews, setPreviews] = useState<Record<number, ComicPreview>>({});
  const [keepId, setKeepId] = useState<number | null>(null);
  const [loadError, setLoadError] = useState('');
  const [submitError, setSubmitError] = useState('');
  const [isMerging, setIsMerging] = useState(false);

  const mergePair = useMemo(
    () => (conflict ? mergePairFromConflict(conflict) : null),
    [conflict],
  );

  const mergingId = useMemo(() => {
    if (!mergePair || keepId === null) return null;
    return keepId === mergePair.baseId ? mergePair.mergingId : mergePair.baseId;
  }, [keepId, mergePair]);

  const keepPreview = keepId !== null ? previews[keepId] : null;
  const mergePreview = mergingId !== null ? previews[mergingId] : null;

  useEffect(() => {
    if (!isOpen || !conflict || !mergePair) {
      setPreviews({});
      setKeepId(null);
      setLoadError('');
      setSubmitError('');
      return;
    }

    setKeepId(mergePair.baseId);
    let cancelled = false;
    setLoadError('');
    void (async () => {
      const [base, merging] = await Promise.all([
        fetchComicById(mergePair.baseId),
        fetchComicById(mergePair.mergingId),
      ]);
      if (cancelled) return;
      if (!base || !merging) {
        setLoadError('Could not load both comics. Try again or merge from the action rail.');
        return;
      }
      setPreviews({
        [base.id]: toPreview(base),
        [merging.id]: toPreview(merging),
      });
    })();

    return () => {
      cancelled = true;
    };
  }, [conflict, isOpen, mergePair]);

  const handleSwap = () => {
    if (!mergePair) return;
    setKeepId((current) => (
      current === mergePair.baseId ? mergePair.mergingId : mergePair.baseId
    ));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!mergePair || keepId === null || mergingId === null) return;
    setSubmitError('');
    setIsMerging(true);
    const resultMsg = await mergeComic(keepId, mergingId);
    setIsMerging(false);
    if (resultMsg !== '') {
      setSubmitError(resultMsg);
      return;
    }
    onMerged(mergingId);
    if (onRetry) {
      await onRetry();
    }
    onClose();
  };

  const handleDismiss = () => {
    if (onDismiss) onDismiss();
    else onClose();
  };

  if (!conflict) return null;

  const headline = conflictHeadline(conflict.code);
  const matchLabel = matchKindLabel(conflict.matchKind);

  return (
    <Modal hasCloseBtn isOpen={isOpen} onClose={handleDismiss} size="default">
      <form className="comic-modal-form" onSubmit={handleSubmit}>
        <header className="modal-form-header">
          <h2>{headline}</h2>
          <p>
            {conflict.message}
            {' '}
            Review the two records below. By default we keep the lower ID; use Swap direction
            if the other row has the better title or metadata.
          </p>
        </header>

        <p className="discovery-conflict-meta">
          <span className="discovery-conflict-chip">{matchLabel}</span>
        </p>

        {loadError ? (
          <p className="form-feedback form-feedback-error" role="alert">
            {loadError}
          </p>
        ) : null}

        {submitError ? (
          <p className="form-feedback form-feedback-error" role="alert">
            {submitError}
          </p>
        ) : null}

        {keepPreview && mergePreview ? (
          <>
            <div className="discovery-conflict-grid" aria-label="Comics to merge">
              <article className="discovery-conflict-card discovery-conflict-card-base">
                <p className="discovery-conflict-card-label">Keep (base)</p>
                <h3 className="discovery-conflict-card-title">{keepPreview.title}</h3>
                <dl className="discovery-conflict-stats">
                  <div><dt>ID</dt><dd>{keepPreview.id}</dd></div>
                  <div><dt>Chapter</dt><dd>{keepPreview.chapter}</dd></div>
                  <div><dt>Rating</dt><dd>{keepPreview.rating}/5</dd></div>
                  <div><dt>Tracked</dt><dd>{keepPreview.track ? 'Yes' : 'No'}</dd></div>
                </dl>
              </article>
              <div className="discovery-conflict-arrow" aria-hidden="true">←</div>
              <article className="discovery-conflict-card">
                <p className="discovery-conflict-card-label">Merge in</p>
                <h3 className="discovery-conflict-card-title">{mergePreview.title}</h3>
                <dl className="discovery-conflict-stats">
                  <div><dt>ID</dt><dd>{mergePreview.id}</dd></div>
                  <div><dt>Chapter</dt><dd>{mergePreview.chapter}</dd></div>
                  <div><dt>Rating</dt><dd>{mergePreview.rating}/5</dd></div>
                  <div><dt>Tracked</dt><dd>{mergePreview.track ? 'Yes' : 'No'}</dd></div>
                </dl>
              </article>
            </div>
            <div className="discovery-conflict-swap-row">
              <button
                type="button"
                className="basic-button reverse-button discovery-conflict-swap"
                onClick={handleSwap}
                disabled={isMerging}
              >
                Swap direction
              </button>
              <p className="discovery-conflict-swap-hint">
                Merging into ID {keepPreview.id}; record {mergePreview.id} will be removed.
              </p>
            </div>
          </>
        ) : null}

        <div className="form-actions">
          <button
            className="basic-button reverse-button"
            type="button"
            onClick={handleDismiss}
            disabled={isMerging}
          >
            Not now
          </button>
          <button
            className="basic-button"
            type="submit"
            disabled={isMerging || !mergePair || !keepPreview || !mergePreview}
          >
            {isMerging ? 'Merging…' : 'Merge & continue'}
          </button>
        </div>
      </form>
    </Modal>
  );
};

export default DiscoveryConflictModal;
