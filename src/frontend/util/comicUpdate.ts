import { apiErrorMessage, authorizedFetch } from '../api/client';
import type { Comic } from '../components/Comics/types';
import config from './Config';
import {
  type CatalogConflictPayload,
  parseCatalogConflictPayload,
} from './catalogConflict';

const SERVER = config.SERVER;

export type ComicUpdateResult =
  | { ok: true; comic?: Comic }
  | { ok: false; kind: 'network' }
  | { ok: false; kind: 'server'; status: number; message: string }
  | { ok: false; kind: 'conflict'; conflict: CatalogConflictPayload };

export const patchComic = async (
  comicId: number,
  body: Record<string, unknown>,
  server = SERVER,
): Promise<ComicUpdateResult> => {
  try {
    const response = await authorizedFetch(
      `/comics/${comicId}`,
      {
        method: 'PUT',
        body: JSON.stringify(body),
      },
      server,
    );
    const contentType = response.headers.get('content-type') ?? '';
    const payload = contentType.includes('application/json')
      ? await response.json().catch(() => null)
      : await response.text();

    if (response.ok) {
      if (payload !== null && typeof payload === 'object' && typeof (payload as Comic).id === 'number') {
        return { ok: true, comic: payload as Comic };
      }
      return { ok: true };
    }

    const conflict = parseCatalogConflictPayload(payload, response.status, comicId);
    if (conflict) {
      return { ok: false, kind: 'conflict', conflict };
    }

    const message = apiErrorMessage(payload, `Update failed (${response.status})`);
    return { ok: false, kind: 'server', status: response.status, message };
  } catch {
    return { ok: false, kind: 'network' };
  }
};

export const fetchComicById = async (
  comicId: number,
  server = SERVER,
): Promise<Comic | null> => {
  try {
    const response = await fetch(`${server}/comics/${comicId}`, {
      method: 'GET',
      headers: { accept: 'application/json' },
    });
    if (!response.ok) return null;
    const data = await response.json();
    if (data === null || typeof data !== 'object' || typeof data.id !== 'number') {
      return null;
    }
    return data as Comic;
  } catch {
    return null;
  }
};
