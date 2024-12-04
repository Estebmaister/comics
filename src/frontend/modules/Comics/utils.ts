import { SetURLSearchParams } from 'react-router-dom';
import { COMIC_CARD_WIDTH, MAIN_PAGE_PADDING } from './constants';

/**
 * Calculate the number of comics that can fit in one row based on window width
 */
export const calculateInlineComics = (): number => {
  const width = window.innerWidth;
  const inlineComicsWOP = Math.floor(width / COMIC_CARD_WIDTH);
  return width - (COMIC_CARD_WIDTH * inlineComicsWOP + MAIN_PAGE_PADDING) >= 0
    ? inlineComicsWOP
    : inlineComicsWOP - 1;
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
