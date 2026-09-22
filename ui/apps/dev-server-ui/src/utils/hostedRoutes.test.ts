import { describe, expect, it } from 'vitest';
import { publicHref } from '../../../../packages/components/src/Link/usePublicHref';

describe('links under /dev', () => {
  it('preserves deep links and their query strings for new tabs', () => {
    expect(publicHref('/run?runID=abc', '/dev')).toBe('/dev/run?runID=abc');
    expect(publicHref('/apps/app?id=abc', '/dev/')).toBe(
      '/dev/apps/app?id=abc',
    );
  });
  it('does not duplicate the prefix or change external destinations', () => {
    for (const href of [
      '/dev/apps',
      'https://example.com',
      '//example.com',
      '#step',
    ]) {
      expect(publicHref(href, '/dev')).toBe(href);
    }
    expect(publicHref('/apps', '/')).toBe('/apps');
  });
});
