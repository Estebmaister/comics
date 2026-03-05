// UI Constants
export const COMIC_SEARCH_PLACEHOLDER = 'Search by comic name';

// Layout Constants
export const COMIC_CARD_WIDTH = 490;
export const MAIN_PAGE_PADDING = 30;
export const COMIC_CARD_ESTIMATED_ROW_HEIGHT = 220;
export const VIEWPORT_RESERVED_HEIGHT = 140;

// Time Constants (in milliseconds)
export const REFRESH_INTERVAL = 3 * 60 * 1000; // 3 minutes

// Button Text
export const BUTTON_TEXT = {
  noFilter: 'No filter',
  unchecked: 'Unchecked',
  all: (total: number) => `All > (${total})`,
  tracked: (total: number) => `Tracked < (${total})`
} as const;
