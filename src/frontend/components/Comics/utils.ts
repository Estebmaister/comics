import { SetURLSearchParams } from 'react-router-dom';
import {
  COMIC_CARD_ESTIMATED_ROW_HEIGHT,
  DESKTOP_TWO_COLUMN_MIN_WIDTH,
  DESKTOP_THREE_COLUMN_MIN_WIDTH,
  MOBILE_COMIC_CARD_ESTIMATED_ROW_HEIGHT,
  MOBILE_LIST_TOP_PADDING_PX,
  MOBILE_NAV_HEIGHT_FALLBACK_PX,
  MOBILE_PAGE_LIMIT_BUFFER_PX,
  PHONE_VIEWPORT_MAX_WIDTH,
  VIEWPORT_RESERVED_HEIGHT,
} from './constants';

export type PageLimitOptions = {
  navHeightPx?: number;
};

/**
 * Parse --nav-height from the document (px from ResizeObserver or rem fallback).
 */
export function readNavHeightPx(): number {
  if (typeof document === 'undefined') {
    return MOBILE_NAV_HEIGHT_FALLBACK_PX;
  }
  const raw = getComputedStyle(document.documentElement).getPropertyValue('--nav-height').trim();
  if (!raw) {
    return MOBILE_NAV_HEIGHT_FALLBACK_PX;
  }
  if (raw.endsWith('px')) {
    const parsed = Number.parseFloat(raw);
    return Number.isFinite(parsed) ? parsed : MOBILE_NAV_HEIGHT_FALLBACK_PX;
  }
  if (raw.endsWith('rem')) {
    const parsed = Number.parseFloat(raw);
    return Number.isFinite(parsed) ? parsed * 16 : MOBILE_NAV_HEIGHT_FALLBACK_PX;
  }
  const parsed = Number.parseFloat(raw);
  return Number.isFinite(parsed) ? parsed : MOBILE_NAV_HEIGHT_FALLBACK_PX;
}

/**
 * Calculate the number of comics that can fit in one row based on the
 * premium-card desktop thresholds.
 */
export const calculateInlineComics = (width = window.innerWidth): number => {
  if (width >= DESKTOP_THREE_COLUMN_MIN_WIDTH) return 3;
  if (width >= DESKTOP_TWO_COLUMN_MIN_WIDTH) return 2;
  return 1;
};

function mobilePageLimit(height: number, navHeightPx: number): number {
  const usable = Math.max(
    0,
    height - navHeightPx - MOBILE_LIST_TOP_PADDING_PX - MOBILE_PAGE_LIMIT_BUFFER_PX,
  );
  const fractionalRows = usable / MOBILE_COMIC_CARD_ESTIMATED_ROW_HEIGHT;
  return fractionalRows >= 2.5 ? 3 : 2;
}

/**
 * Calculate page size using both width and height.
 * - 3 columns => 9 cards (3×3 grid)
 * - 2 columns => height-dependent row count
 * - 1 column => 2–3 cards on narrow viewports
 */
export const calculatePageLimit = (
  width = window.innerWidth,
  height = window.innerHeight,
  options: PageLimitOptions = {},
): number => {
  const columns = calculateInlineComics(width);
  if (columns >= 3) return 9;
  if (columns === 1) {
    if (width > PHONE_VIEWPORT_MAX_WIDTH) return 3;

    const navHeightPx = options.navHeightPx ?? readNavHeightPx();
    return mobilePageLimit(height, navHeightPx);
  }

  const usableHeight = Math.max(0, height - VIEWPORT_RESERVED_HEIGHT);
  const possibleRows = Math.floor(usableHeight / COMIC_CARD_ESTIMATED_ROW_HEIGHT);
  const rows = Math.max(2, Math.min(4, possibleRows));
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
