import { SetURLSearchParams } from 'react-router-dom';
import {
  COMIC_CARD_ESTIMATED_ROW_HEIGHT,
  DESKTOP_TWO_COLUMN_MIN_WIDTH,
  DESKTOP_THREE_COLUMN_MIN_WIDTH,
  VIEWPORT_RESERVED_HEIGHT,
} from './constants';

/**
 * Calculate the number of comics that can fit in one row based on the
 * premium-card desktop thresholds.
 */
export const calculateInlineComics = (width = window.innerWidth): number => {
  if (width >= DESKTOP_THREE_COLUMN_MIN_WIDTH) return 3;
  if (width >= DESKTOP_TWO_COLUMN_MIN_WIDTH) return 2;
  return 1;
};

/**
 * Calculate page size using both width and height.
 * - 3 columns => 9 cards
 * - 2 columns => 6 or 8 cards (height dependent, always even)
 * - 1 column => 3 cards
 */
export const calculatePageLimit = (
  width = window.innerWidth,
  height = window.innerHeight
): number => {
  const columns = calculateInlineComics(width);
  if (columns >= 3) return 9;
  if (columns === 1) return 3;

  const usableHeight = Math.max(0, height - VIEWPORT_RESERVED_HEIGHT);
  const possibleRows = Math.floor(usableHeight / COMIC_CARD_ESTIMATED_ROW_HEIGHT);
  const rows = Math.max(3, Math.min(4, possibleRows));
  return columns * rows;
};

/**
 * Handle toggling the unchecked comics filter
 */
export const handleOnlyUnchecked = (
  setSearchParams: SetURLSearchParams,
  onlyUnchecked: boolean
) => () => {
  setSearchParams(prev => {
    prev.set('onlyUnchecked', String(!onlyUnchecked));
    prev.delete('from');
    if (onlyUnchecked) prev.delete('onlyUnchecked');
    return prev;
  }, { replace: false });
};

/**
 * Handle toggling the tracked comics filter
 */
export const handleOnlyTracked = (
  setSearchParams: SetURLSearchParams,
  onlyTracked: boolean
) => () => {
  setSearchParams(prev => {
    prev.set('onlyTracked', String(!onlyTracked));
    prev.delete('from');
    if (onlyTracked) {
      prev.delete('onlyTracked');
      prev.delete('onlyUnchecked');
    }
    return prev;
  }, { replace: false });
};

/**
 * Handle search input changes
 */
export const handleSearchInput = (
  setSearchParams: SetURLSearchParams,
  value?: string
) => {
  setSearchParams(prev => {
    if (value !== undefined) {
      prev.set('queryFilter', value);
    }
    prev.set('from', '0');
    return prev;
  }, { replace: true });
};
