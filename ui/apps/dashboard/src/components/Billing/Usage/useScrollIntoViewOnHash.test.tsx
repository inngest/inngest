// @vitest-environment jsdom
import { cleanup, render } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';

import { useScrollIntoViewOnHash } from './useScrollIntoViewOnHash';

const scrollIntoView = vi.fn();

function Section({ loaded }: { loaded: boolean }) {
  const ref = useScrollIntoViewOnHash('section');
  return loaded ? <div ref={ref} /> : null;
}

beforeEach(() => {
  Element.prototype.scrollIntoView = scrollIntoView;
});

afterEach(() => {
  cleanup();
  scrollIntoView.mockReset();
  window.location.hash = '';
});

it('scrolls once the section renders after data loads', () => {
  window.location.hash = '#section';
  const { rerender } = render(<Section loaded={false} />);
  expect(scrollIntoView).not.toHaveBeenCalled();

  rerender(<Section loaded />);
  rerender(<Section loaded={false} />);
  rerender(<Section loaded />);
  expect(scrollIntoView).toHaveBeenCalledTimes(1);
});

it('does not scroll for a different hash', () => {
  window.location.hash = '#other';
  render(<Section loaded />);
  expect(scrollIntoView).not.toHaveBeenCalled();
});
