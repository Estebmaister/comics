import { useEffect, useRef } from 'react';
import type { Comic } from '../types';
import { useDiscoveryConflict } from './DiscoveryConflictContext';
import {
  dismissDuplicatePair,
  findCatalogDuplicatePairs,
  isDuplicatePairDismissed,
} from '../../../util/catalogDuplicatePairs';

type CatalogDuplicateWatcherProps = {
  comics: Comic[];
  enabled: boolean;
  viewKey: string;
};

const CatalogDuplicateWatcher = ({
  comics,
  enabled,
  viewKey,
}: CatalogDuplicateWatcherProps) => {
  const discovery = useDiscoveryConflict();
  const promptedForViewRef = useRef('');

  useEffect(() => {
    if (!enabled || !discovery || comics.length < 2) return;

    const pairs = findCatalogDuplicatePairs(comics).filter(
      (pair) => !isDuplicatePairDismissed(pair.baseId, pair.mergingId),
    );
    if (pairs.length === 0) return;

    const best = pairs[0];
    const promptKey = `${viewKey}:${best.baseId}:${best.mergingId}`;
    if (promptedForViewRef.current === promptKey) return;
    promptedForViewRef.current = promptKey;

    discovery.openCatalogConflict({
      conflict: {
        code: 'duplicate_comic_match',
        sourceComicId: best.baseId,
        ownerComicId: best.mergingId,
        candidateIds: [],
        matchKind: best.matchKind,
        message: 'These entries on this page look like the same series.',
      },
      notify: true,
      onDismiss: () => dismissDuplicatePair(best.baseId, best.mergingId),
    });
  }, [comics, discovery, enabled, viewKey]);

  return null;
};

export default CatalogDuplicateWatcher;
