import { describe, expect, it } from 'vitest';

import { pathCreator } from './pathCreator';

describe('pathCreator', () => {
  describe('function', () => {
    it('links to the function config slideover for the given slug', () => {
      expect(pathCreator.function({ functionSlug: 'my-fn' })).toBe(
        '/functions/config?slug=my-fn',
      );
    });

    it('encodes special characters in the slug', () => {
      expect(pathCreator.function({ functionSlug: 'my fn/slug' })).toBe(
        '/functions/config?slug=my+fn%2Fslug',
      );
    });
  });
});
