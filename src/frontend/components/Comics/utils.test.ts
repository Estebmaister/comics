import { calculateInlineComics, calculatePageLimit } from './utils';

test('uses the premium desktop width thresholds for card density', () => {
  expect(calculateInlineComics(899)).toBe(1);
  expect(calculateInlineComics(900)).toBe(2);
  expect(calculateInlineComics(1599)).toBe(2);
  expect(calculateInlineComics(1600)).toBe(3);
});

test('keeps pagination sizes aligned with the hybrid layout', () => {
  expect(calculatePageLimit(800, 900)).toBe(3);
  expect(calculatePageLimit(1200, 900)).toBe(6);
  expect(calculatePageLimit(1200, 1300)).toBe(8);
  expect(calculatePageLimit(1700, 900)).toBe(9);
});
