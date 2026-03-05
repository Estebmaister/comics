import { SetURLSearchParams } from 'react-router-dom';
import {
  COMIC_CARD_ESTIMATED_ROW_HEIGHT,
  COMIC_CARD_WIDTH,
  MAIN_PAGE_PADDING,
  VIEWPORT_RESERVED_HEIGHT,
} from './constants';

/**
 * Calculate the number of comics that can fit in one row based on window width
 */
export const calculateInlineComics = (width = window.innerWidth): number => {
  const inlineComicsWOP = Math.floor(width / COMIC_CARD_WIDTH);
  if (inlineComicsWOP <= 0 || width <= 0) return 1;
  const inlineComics = width - (COMIC_CARD_WIDTH * inlineComicsWOP + MAIN_PAGE_PADDING) >= 0
    ? inlineComicsWOP
    : inlineComicsWOP - 1;
  // Keep pagination predictable: 3 columns max.
  return Math.min(3, Math.max(1, inlineComics));
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
