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
} from './constants';

const DESKTOP_MIN_ROWS = 2;
const DESKTOP_MAX_ROWS = 4;
/** When ~2.15+ desktop card rows fit, prefer 3 rows (matches ~900px-tall viewports). */
const DESKTOP_THREE_ROW_FRACTION = 2.15;
/** Need ~3.75+ fractional rows before using a fourth row. */
const DESKTOP_FOUR_ROW_FRACTION = 3.75;

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

function listViewportMargin(navHeightPx: number): number {
  return navHeightPx + MOBILE_LIST_TOP_PADDING_PX + MOBILE_PAGE_LIMIT_BUFFER_PX;
}

function mobilePageLimit(height: number, navHeightPx: number): number {
  const usable = Math.max(0, height - listViewportMargin(navHeightPx));
  const fractionalRows = usable / MOBILE_COMIC_CARD_ESTIMATED_ROW_HEIGHT;
  return fractionalRows >= 2.5 ? 3 : 2;
}

/**
 * Desktop / tablet row count from viewport height (19.1rem cards + grid gap).
 */
export function desktopRowCount(height: number, navHeightPx: number): number {
  const usable = Math.max(0, height - listViewportMargin(navHeightPx));
  const fractionalRows = usable / COMIC_CARD_ESTIMATED_ROW_HEIGHT;

  if (fractionalRows < DESKTOP_THREE_ROW_FRACTION) {
    return DESKTOP_MIN_ROWS;
  }
  if (fractionalRows < DESKTOP_FOUR_ROW_FRACTION) {
    return 3;
  }
  return DESKTOP_MAX_ROWS;
}

/**
 * Calculate page size using both width and height (columns × rows).
 * Phone (≤640px) uses shorter mobile card stride; wider layouts use desktop cards.
 */
export const calculatePageLimit = (
  width = window.innerWidth,
  height = window.innerHeight,
  options: PageLimitOptions = {},
): number => {
  const columns = calculateInlineComics(width);
  const navHeightPx = options.navHeightPx ?? readNavHeightPx();

  if (columns === 1 && width <= PHONE_VIEWPORT_MAX_WIDTH) {
    return mobilePageLimit(height, navHeightPx);
  }

  const rows = desktopRowCount(height, navHeightPx);
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
