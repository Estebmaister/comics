import type { Comic } from '../components/Comics/types';

export type CatalogDuplicatePair = {
  baseId: number;
  mergingId: number;
  matchKind: 'title_prefix' | 'exact_title';
  score: number;
};

const normalizeTitle = (title: string): string => (
  title
    .toLowerCase()
    .replace(/[''`]/g, '')
    .replace(/[^a-z0-9]+/g, ' ')
    .trim()
    .replace(/\s+/g, ' ')
);

const primaryTitle = (comic: Comic): string => comic.titles[0]?.trim() ?? '';

const titleRelation = (left: string, right: string): CatalogDuplicatePair['matchKind'] | null => {
  const a = normalizeTitle(left);
  const b = normalizeTitle(right);
  if (!a || !b) return null;
  if (a === b) return 'exact_title';
  const shorter = a.length <= b.length ? a : b;
  const longer = a.length <= b.length ? b : a;
  if (shorter.length < 12) return null;
  if (longer.startsWith(shorter)) return 'title_prefix';
  return null;
};

const chaptersClose = (left: Comic, right: Comic): boolean => {
  const spread = Math.abs(left.current_chap - right.current_chap);
  if (spread <= 15) return true;
  const max = Math.max(left.current_chap, right.current_chap);
  if (max < 20) return spread <= 2;
  const min = Math.min(left.current_chap, right.current_chap);
  return min / max >= 0.85;
};

const pairScore = (kind: CatalogDuplicatePair['matchKind'], spread: number): number => {
  const base = kind === 'exact_title' ? 100 : 80;
  return base - Math.min(spread, 40);
};

/** Find likely duplicate pairs visible on one catalog page (admin merge suggestions). */
export const findCatalogDuplicatePairs = (comics: Comic[]): CatalogDuplicatePair[] => {
  const pairs: CatalogDuplicatePair[] = [];
  for (let i = 0; i < comics.length; i += 1) {
    for (let j = i + 1; j < comics.length; j += 1) {
      const left = comics[i];
      const right = comics[j];
      if (left.com_type !== right.com_type) continue;
      if (!chaptersClose(left, right)) continue;

      let bestKind: CatalogDuplicatePair['matchKind'] | null = null;
      for (const leftTitle of left.titles) {
        for (const rightTitle of right.titles) {
          const kind = titleRelation(leftTitle, rightTitle);
          if (!kind) continue;
          if (bestKind === 'exact_title') break;
          bestKind = kind;
        }
        if (bestKind === 'exact_title') break;
      }
      if (!bestKind) continue;

      const baseId = Math.min(left.id, right.id);
      const mergingId = Math.max(left.id, right.id);
      const spread = Math.abs(left.current_chap - right.current_chap);
      pairs.push({
        baseId,
        mergingId,
        matchKind: bestKind,
        score: pairScore(bestKind, spread),
      });
    }
  }

  pairs.sort((a, b) => b.score - a.score);
  const seen = new Set<string>();
  return pairs.filter((pair) => {
    const key = `${pair.baseId}:${pair.mergingId}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
};

const dismissKey = (baseId: number, mergingId: number) => (
  `comics:dismissed-duplicate:${baseId}:${mergingId}`
);

export const isDuplicatePairDismissed = (baseId: number, mergingId: number): boolean => (
  sessionStorage.getItem(dismissKey(baseId, mergingId)) === '1'
);

export const dismissDuplicatePair = (baseId: number, mergingId: number): void => {
  sessionStorage.setItem(dismissKey(baseId, mergingId), '1');
};
