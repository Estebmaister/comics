// UI Constants
export const COMIC_SEARCH_PLACEHOLDER = 'Search by comic name';

// Layout Constants
export const COMIC_CARD_WIDTH = 450;
export const MAIN_PAGE_PADDING = 30;
export const COMICS_PER_ROW = 3;

// Time Constants (in milliseconds)
export const REFRESH_INTERVAL = 3 * 60 * 1000; // 3 minutes

// Button Text
export const BUTTON_TEXT = {
  noFilter: 'No filter',
  unchecked: 'Unchecked',
  all: (total: number) => `All > (${total})`,
  tracked: (total: number) => `Tracked < (${total})`
} as const;

// CSS Classes
export const CSS_CLASSES = {
  navBar: 'nav-bar',
  searchBox: 'search-box',
  serverMessage: 'server',
  comicList: 'comic-list',
  basicButton: 'basic-button',
  reverseButton: 'reverse-button',
  allTrackButton: 'all-track-button',
  barButton: 'bar-button'
} as const;
