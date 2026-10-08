import { useEffect, useMemo, useState } from 'react';
import type { SetURLSearchParams } from 'react-router-dom';
import Modal from '../../Modal';

export type SortBy = 'last_update' | 'rating' | 'id';
export type SortDir = 'desc' | 'asc';

export type SortFilterState = {
  ratingMin?: number;
  ratingMax?: number;
  sortBy: SortBy;
  sortDir: SortDir;
};

const clampRating = (value: unknown) => {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return undefined;
  const asInt = Math.trunc(parsed);
  if (asInt < 0 || asInt > 5) return undefined;
  return asInt;
};

const toOptionalNumberParam = (value?: number) => (
  value === undefined ? undefined : String(value)
);

export function SortFilterModal({
  isOpen,
  onClose,
  setSearchParams,
  initialState,
}: {
  isOpen: boolean;
  onClose: () => void;
  setSearchParams: SetURLSearchParams;
  initialState: SortFilterState;
}) {
  const [ratingMin, setRatingMin] = useState<number | undefined>(initialState.ratingMin);
  const [ratingMax, setRatingMax] = useState<number | undefined>(initialState.ratingMax);
  const [sortBy, setSortBy] = useState<SortBy>(initialState.sortBy);
  const [sortDir, setSortDir] = useState<SortDir>(initialState.sortDir);

  useEffect(() => {
    if (!isOpen) return;
    setRatingMin(initialState.ratingMin);
    setRatingMax(initialState.ratingMax);
    setSortBy(initialState.sortBy);
    setSortDir(initialState.sortDir);
  }, [isOpen, initialState.ratingMin, initialState.ratingMax, initialState.sortBy, initialState.sortDir]);

  const hasChanges = useMemo(() => (
    ratingMin !== initialState.ratingMin
    || ratingMax !== initialState.ratingMax
    || sortBy !== initialState.sortBy
    || sortDir !== initialState.sortDir
  ), [initialState, ratingMax, ratingMin, sortBy, sortDir]);

  const apply = () => {
    const normalizedMin = clampRating(ratingMin);
    const normalizedMax = clampRating(ratingMax);
    const effectiveMin = normalizedMin;
    const effectiveMax = normalizedMax !== undefined && effectiveMin !== undefined && normalizedMax < effectiveMin
      ? effectiveMin
      : normalizedMax;

    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete('from');

      const minParam = toOptionalNumberParam(effectiveMin);
      const maxParam = toOptionalNumberParam(effectiveMax);
      if (minParam === undefined) next.delete('ratingMin');
      else next.set('ratingMin', minParam);
      if (maxParam === undefined) next.delete('ratingMax');
      else next.set('ratingMax', maxParam);

      if (sortBy === 'last_update') next.delete('sortBy');
      else next.set('sortBy', sortBy);

      if (sortDir === 'desc') next.delete('sortDir');
      else next.set('sortDir', sortDir);

      return next;
    }, { replace: false });
    onClose();
  };

  const reset = () => {
    setRatingMin(undefined);
    setRatingMax(undefined);
    setSortBy('last_update');
    setSortDir('desc');
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="compact" hasCloseBtn={true}>
      <form className="comic-modal-form" onSubmit={(e) => { e.preventDefault(); apply(); }}>
        <header className="modal-form-header">
          <h2>Sort &amp; filter</h2>
          <p>Keep your list tidy without crowding the toolbar.</p>
        </header>

        <div className="form-grid" style={{ gridTemplateColumns: '1fr', gap: '1rem' }}>
          <div className="form-row">
            <label htmlFor="ratingMin">Rating min (0–5)</label>
            <input
              className="app-field"
              id="ratingMin"
              name="ratingMin"
              type="number"
              min={0}
              max={5}
              value={ratingMin ?? ''}
              onChange={(e) => setRatingMin(clampRating(e.target.value))}
            />
          </div>

          <div className="form-row">
            <label htmlFor="ratingMax">Rating max (0–5)</label>
            <input
              className="app-field"
              id="ratingMax"
              name="ratingMax"
              type="number"
              min={0}
              max={5}
              value={ratingMax ?? ''}
              onChange={(e) => setRatingMax(clampRating(e.target.value))}
            />
          </div>

          <div className="form-row">
            <label htmlFor="sortBy">Sort by</label>
            <select
              className="app-field"
              id="sortBy"
              name="sortBy"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as SortBy)}
            >
              <option value="last_update">Last update</option>
              <option value="rating">Rating</option>
              <option value="id">ID</option>
            </select>
          </div>

          <div className="form-row">
            <label htmlFor="sortDir">Sort direction</label>
            <select
              className="app-field"
              id="sortDir"
              name="sortDir"
              value={sortDir}
              onChange={(e) => setSortDir(e.target.value as SortDir)}
            >
              <option value="desc">Descending</option>
              <option value="asc">Ascending</option>
            </select>
          </div>
        </div>

        <div className="form-actions" style={{ display: 'flex', gap: '0.75rem', justifyContent: 'space-between' }}>
          <button className="basic-button neutral-button" type="button" onClick={reset} disabled={!hasChanges}>
            Reset
          </button>
          <button className="basic-button" type="submit">
            Apply
          </button>
        </div>
      </form>
    </Modal>
  );
}

