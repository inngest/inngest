import assert from 'node:assert';
import { describe, it } from 'vitest';

import { getVisiblePages, makeInclusiveRange } from './getVisiblePages';

describe('getVisiblePages', () => {
  describe('normal variant (7 slots - DEFAULT)', () => {
    it('shows all pages when total <= 7', () => {
      assert.deepEqual(getVisiblePages({ current: 1, total: 1, variant: 'normal' }), [1]);
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 2, variant: 'normal' }),
        makeInclusiveRange(1, 2)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 3, variant: 'normal' }),
        makeInclusiveRange(1, 3)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 4, variant: 'normal' }),
        makeInclusiveRange(1, 4)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 5, variant: 'normal' }),
        makeInclusiveRange(1, 5)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 6, variant: 'normal' }),
        makeInclusiveRange(1, 6)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 7, variant: 'normal' }),
        makeInclusiveRange(1, 7)
      );
    });

    it('beginning pattern', () => {
      assert.deepEqual(getVisiblePages({ current: 1, total: 10, variant: 'normal' }), [
        ...makeInclusiveRange(1, 5),
        '...',
        10,
      ]);
      assert.deepEqual(getVisiblePages({ current: 2, total: 10, variant: 'normal' }), [
        ...makeInclusiveRange(1, 5),
        '...',
        10,
      ]);
      assert.deepEqual(getVisiblePages({ current: 3, total: 10, variant: 'normal' }), [
        ...makeInclusiveRange(1, 5),
        '...',
        10,
      ]);
      assert.deepEqual(getVisiblePages({ current: 4, total: 10, variant: 'normal' }), [
        ...makeInclusiveRange(1, 5),
        '...',
        10,
      ]);
    });

    it('middle pattern', () => {
      assert.deepEqual(getVisiblePages({ current: 5, total: 10, variant: 'normal' }), [
        1,
        '...',
        ...makeInclusiveRange(4, 6),
        '...',
        10,
      ]);
      assert.deepEqual(getVisiblePages({ current: 6, total: 10, variant: 'normal' }), [
        1,
        '...',
        ...makeInclusiveRange(5, 7),
        '...',
        10,
      ]);
    });

    it('end pattern', () => {
      assert.deepEqual(getVisiblePages({ current: 7, total: 10, variant: 'normal' }), [
        1,
        '...',
        ...makeInclusiveRange(6, 10),
      ]);
      assert.deepEqual(getVisiblePages({ current: 8, total: 10, variant: 'normal' }), [
        1,
        '...',
        ...makeInclusiveRange(6, 10),
      ]);
      assert.deepEqual(getVisiblePages({ current: 9, total: 10, variant: 'normal' }), [
        1,
        '...',
        ...makeInclusiveRange(6, 10),
      ]);
      assert.deepEqual(getVisiblePages({ current: 10, total: 10, variant: 'normal' }), [
        1,
        '...',
        ...makeInclusiveRange(6, 10),
      ]);
    });
  });

  describe('narrow variant (5 slots)', () => {
    it('shows all pages when total <= 5', () => {
      assert.deepEqual(getVisiblePages({ current: 1, total: 1, variant: 'narrow' }), [1]);
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 2, variant: 'narrow' }),
        makeInclusiveRange(1, 2)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 3, variant: 'narrow' }),
        makeInclusiveRange(1, 3)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 4, variant: 'narrow' }),
        makeInclusiveRange(1, 4)
      );
      assert.deepEqual(
        getVisiblePages({ current: 1, total: 5, variant: 'narrow' }),
        makeInclusiveRange(1, 5)
      );
    });

    it('beginning pattern', () => {
      assert.deepEqual(getVisiblePages({ current: 1, total: 6, variant: 'narrow' }), [
        ...makeInclusiveRange(1, 3),
        '...',
        6,
      ]);
      assert.deepEqual(getVisiblePages({ current: 2, total: 6, variant: 'narrow' }), [
        ...makeInclusiveRange(1, 3),
        '...',
        6,
      ]);
    });

    it('middle pattern', () => {
      assert.deepEqual(getVisiblePages({ current: 3, total: 6, variant: 'narrow' }), [
        1,
        '...',
        3,
        '...',
        6,
      ]);
      assert.deepEqual(getVisiblePages({ current: 4, total: 6, variant: 'narrow' }), [
        1,
        '...',
        4,
        '...',
        6,
      ]);
    });

    it('end pattern', () => {
      assert.deepEqual(getVisiblePages({ current: 5, total: 6, variant: 'narrow' }), [
        1,
        '...',
        ...makeInclusiveRange(4, 6),
      ]);
      assert.deepEqual(getVisiblePages({ current: 6, total: 6, variant: 'narrow' }), [
        1,
        '...',
        ...makeInclusiveRange(4, 6),
      ]);
    });
  });
});
