import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import ProgressBar from './ProgressBar';

afterEach(cleanup);

describe('ProgressBar', () => {
  it('preserves the default progress bar treatment', () => {
    const { container } = render(<ProgressBar limit={100} value={50} />);

    expect(screen.getByRole('progressbar').className).toContain('h-6');
    expect(container.querySelector('[style="width: 50%;"]')).toBeTruthy();
  });

  it('renders the small error treatment at the limit', () => {
    const { container } = render(<ProgressBar kind="error" limit={100} size="small" value={100} />);

    const progressBar = screen.getByRole('progressbar');
    expect(progressBar.className).toContain('h-1');
    expect(progressBar.className).toContain('bg-tertiary-xSubtle');

    const indicator = container.querySelector('[style="width: 100%;"]');
    expect(indicator?.className).toContain('bg-tertiary-intense');
  });
});
