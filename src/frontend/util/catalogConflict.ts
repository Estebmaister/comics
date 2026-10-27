export type CatalogConflictCode =
  | 'discovery_key_conflict'
  | 'duplicate_comic_match'
  | 'ambiguous_comic_match';

export type CatalogConflictPayload = {
  code: CatalogConflictCode;
  sourceComicId: number;
  ownerComicId?: number;
  candidateIds: number[];
  matchKind?: string;
  message: string;
};

const readInt = (value: unknown): number | undefined => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return Math.trunc(parsed);
  }
  return undefined;
};

const readIntList = (value: unknown): number[] => {
  if (!Array.isArray(value)) return [];
  return value
    .map((item) => readInt(item))
    .filter((item): item is number => item !== undefined);
};

export const parseCatalogConflictPayload = (
  payload: unknown,
  status: number,
  sourceComicId: number,
): CatalogConflictPayload | null => {
  if (status !== 409 || payload === null || typeof payload !== 'object') {
    return null;
  }
  const body = payload as Record<string, unknown>;
  const code = body.code;
  if (
    code !== 'discovery_key_conflict'
    && code !== 'duplicate_comic_match'
    && code !== 'ambiguous_comic_match'
  ) {
    return null;
  }

  const comicId = readInt(body.comic_id) ?? sourceComicId;
  const ownerComicId = readInt(body.owner_comic_id);
  const candidateIds = readIntList(body.candidate_ids);
  const matchKind = typeof body.match_kind === 'string' ? body.match_kind : undefined;
  const message = typeof body.message === 'string' && body.message.trim() !== ''
    ? body.message
    : 'This update conflicts with another catalog entry.';

  return {
    code,
    sourceComicId: comicId,
    ownerComicId,
    candidateIds,
    matchKind,
    message,
  };
};

/** Pick merge base (kept) and duplicate (removed) using lowest-id rule. */
export const mergePairFromConflict = (
  conflict: CatalogConflictPayload,
): { baseId: number; mergingId: number } | null => {
  const otherId = conflict.ownerComicId
    ?? conflict.candidateIds.find((id) => id !== conflict.sourceComicId);
  if (otherId === undefined) return null;
  const baseId = Math.min(conflict.sourceComicId, otherId);
  const mergingId = Math.max(conflict.sourceComicId, otherId);
  if (baseId === mergingId) return null;
  return { baseId, mergingId };
};

export const conflictHeadline = (code: CatalogConflictCode): string => {
  switch (code) {
    case 'discovery_key_conflict':
      return 'Discovery key conflict';
    case 'ambiguous_comic_match':
      return 'Ambiguous title match';
    default:
      return 'Possible duplicate';
  }
};

export const matchKindLabel = (matchKind?: string): string => {
  if (!matchKind) return 'Title overlap';
  switch (matchKind) {
    case 'discovery_key':
      return 'Exact discovery key';
    case 'title_prefix':
      return 'Title prefix';
    case 'title_tokens':
      return 'Similar title tokens';
    default:
      return matchKind.replaceAll('_', ' ');
  }
};
