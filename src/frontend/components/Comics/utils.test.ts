import { calculateInlineComics, calculatePageLimit } from './utils';
import { MOBILE_COMIC_CARD_ESTIMATED_ROW_HEIGHT } from './constants';

test('uses the premium desktop width thresholds for card density', () => {
  expect(calculateInlineComics(899)).toBe(1);
  expect(calculateInlineComics(900)).toBe(2);
  expect(calculateInlineComics(1599)).toBe(2);
  expect(calculateInlineComics(1600)).toBe(3);
});

test('keeps pagination sizes aligned with the hybrid layout', () => {
  expect(calculatePageLimit(800, 900)).toBe(3);
  expect(calculatePageLimit(1200, 900)).toBe(4);
  expect(calculatePageLimit(1200, 1300)).toBe(6);
  expect(calculatePageLimit(1700, 900)).toBe(9);
});

test('prefers three phone rows when about two and a half cards fit', () => {
  const navPx = 190;
  const stride = MOBILE_COMIC_CARD_ESTIMATED_ROW_HEIGHT;
  const heightForThree = navPx + 10 + 8 + stride * 2.5;

  expect(calculatePageLimit(390, 844, { navHeightPx: navPx })).toBe(3);
  expect(calculatePageLimit(479, 717, { navHeightPx: navPx })).toBe(2);
  expect(calculatePageLimit(390, heightForThree - 20, { navHeightPx: navPx })).toBe(2);
});
