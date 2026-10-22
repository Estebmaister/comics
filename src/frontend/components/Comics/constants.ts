// UI Constants
export const COMIC_SEARCH_PLACEHOLDER = 'Search by comic name';

// Layout Constants
export const DESKTOP_TWO_COLUMN_MIN_WIDTH = 900;
export const DESKTOP_THREE_COLUMN_MIN_WIDTH = 1600;
export const PHONE_VIEWPORT_MAX_WIDTH = 640;
// Card grid row stride: fixed card height (19.1rem) + sm grid gap (1rem) at 16px root.
export const COMIC_CARD_ESTIMATED_ROW_HEIGHT = 322;
/** @deprecated Use measured --nav-height + list padding in utils.ts */
export const VIEWPORT_RESERVED_HEIGHT = 180;
// Mobile: 14.1rem card + gap-2.5 (0.625rem) at 16px root.
export const MOBILE_COMIC_CARD_HEIGHT_REM = 14.1;
export const MOBILE_COMIC_GRID_GAP_REM = 0.625;
export const MOBILE_COMIC_CARD_ESTIMATED_ROW_HEIGHT = Math.round(
  (MOBILE_COMIC_CARD_HEIGHT_REM + MOBILE_COMIC_GRID_GAP_REM) * 16,
);
/** Fallback when --nav-height is still rem-based (before measure). */
export const MOBILE_NAV_HEIGHT_FALLBACK_PX = 176;
export const MOBILE_LIST_TOP_PADDING_PX = 10;
export const MOBILE_PAGE_LIMIT_BUFFER_PX = 8;
export const MOBILE_VIEWPORT_RESERVED_HEIGHT = MOBILE_NAV_HEIGHT_FALLBACK_PX
  + MOBILE_LIST_TOP_PADDING_PX
  + MOBILE_PAGE_LIMIT_BUFFER_PX;

// Time Constants (in milliseconds)
export const REFRESH_INTERVAL = 3 * 60 * 1000; // 3 minutes
export const SEARCH_DEBOUNCE_MS = 300;

// Button Text
export const BUTTON_TEXT = {
  noFilter: 'No filter',
  unchecked: 'Unchecked',
  all: (total: number) => `All > (${total})`,
  tracked: (total: number) => `Tracked < (${total})`
} as const;
