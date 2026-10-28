// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';

import { createTooltipTimestamp } from './tooltipTimestamp';
import { tooltipFormatter } from './utils';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const timestamp = '2026-10-28T16:06:51.123-04:00';
const iso = '2026-10-28T20:06:51.123Z';
const setup = (writeText = vi.fn().mockResolvedValue(undefined)) => {
  vi.stubGlobal('navigator', { clipboard: { writeText } });
  const header = createTooltipTimestamp(timestamp);
  const button = header.querySelector('button')!;
  return { header, button, writeText };
};

afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe('tooltip timestamp', () => {
  it.each([
    {
      date: new Date(2026, 9, 28, 16, 6, 51),
      local: '28 Oct 2026, 04:06:51 PM',
    },
    { date: new Date(2027, 0, 1, 0, 0, 0), local: '01 Jan 2027, 12:00:00 AM' },
    {
      date: new Date(2026, 6, 1, 12, 30, 9),
      local: '01 Jul 2026, 12:30:09 PM',
    },
  ])('shows local and UTC times for $local', ({ date, local }) => {
    const header = createTooltipTimestamp(date.toISOString());
    const times = header.querySelectorAll('time');
    const utcDate = date.toISOString();
    const utcMonth = [
      'Jan',
      'Feb',
      'Mar',
      'Apr',
      'May',
      'Jun',
      'Jul',
      'Aug',
      'Sep',
      'Oct',
      'Nov',
      'Dec',
    ][date.getUTCMonth()];
    const utc = `${utcDate.slice(8, 10)} ${utcMonth} ${date.getUTCFullYear()}, ${utcDate.slice(11, 19)}`;

    expect(times[0]?.textContent).toBe(local);
    expect(times[1]?.textContent).toBe(utc);
    expect([...times].map((time) => time.dateTime)).toEqual([utcDate, utcDate]);
    expect(header.textContent).toContain('Local');
    expect(header.textContent).toContain('UTC');
  });

  it('copies the exact ISO timestamp, including milliseconds, after clipboard success', async () => {
    let finish!: () => void;
    const writeText = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        }),
    );
    const { button } = setup(writeText);

    button.click();
    button.click();
    expect(writeText).toHaveBeenCalledTimes(1);
    expect(writeText).toHaveBeenCalledWith(iso);
    expect(button.getAttribute('aria-disabled')).toBe('true');
    expect(toast.success).not.toHaveBeenCalled();
    expect(button.getAttribute('aria-label')).toBe('Copy timestamp');

    finish();
    await vi.waitFor(() =>
      expect(toast.success).toHaveBeenCalledWith('Copied to clipboard'),
    );
    expect(button.hasAttribute('aria-disabled')).toBe(false);
    expect(button.getAttribute('aria-label')).toBe('Timestamp copied');
  });

  it('reports clipboard failures and allows retrying', async () => {
    const writeText = vi
      .fn()
      .mockRejectedValueOnce(new Error('denied'))
      .mockResolvedValueOnce(undefined);
    const { button } = setup(writeText);

    button.click();
    await vi.waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith('Could not copy timestamp'),
    );
    expect(toast.success).not.toHaveBeenCalled();
    expect(button.hasAttribute('aria-disabled')).toBe(false);
    expect(button.getAttribute('aria-label')).toBe('Copy timestamp');

    button.click();
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledOnce());
  });

  it('handles an unavailable clipboard without claiming success', async () => {
    vi.stubGlobal('navigator', {});
    const button = createTooltipTimestamp(timestamp).querySelector('button')!;
    button.click();
    await vi.waitFor(() => expect(toast.error).toHaveBeenCalledOnce());
    expect(toast.success).not.toHaveBeenCalled();
    expect(button.hasAttribute('aria-disabled')).toBe(false);
  });

  it.each(['No Data Found', '', '<img src=x onerror=alert(1)>'])(
    'renders non-date labels as plain text (%s)',
    (value) => {
      const header = createTooltipTimestamp(value);
      expect(header.textContent).toBe(value);
      expect(header.children).toHaveLength(0);
    },
  );

  it('keeps clipboard values isolated between tooltips', async () => {
    const { button, writeText } = setup();
    const otherButton = createTooltipTimestamp(
      '2026-11-01T06:30:00Z',
    ).querySelector('button')!;
    otherButton.click();
    button.click();
    await vi.waitFor(() => expect(writeText).toHaveBeenCalledTimes(2));
    expect(writeText.mock.calls).toEqual([['2026-11-01T06:30:00.000Z'], [iso]]);
  });
});

describe('metrics tooltip', () => {
  it('renders function names as text rather than HTML', () => {
    const name = '<img src=x onerror=alert(1)> & "quoted"';
    const tooltip = tooltipFormatter([
      { axisValue: timestamp, seriesName: name, value: 1, color: '#000' },
    ]) as HTMLElement;
    expect(tooltip.textContent).toContain(name);
    expect(tooltip.querySelector('img')).toBeNull();
  });

  it('keeps zero values, omits missing buckets, and sorts by value', () => {
    const params = [
      { axisValue: timestamp, seriesName: 'Zero', value: 0, color: '#000' },
      {
        axisValue: timestamp,
        seriesName: 'Missing',
        value: null,
        color: '#000',
      },
      { axisValue: timestamp, seriesName: 'Most', value: 10, color: '#000' },
    ];
    const tooltip = tooltipFormatter(params) as HTMLElement;
    expect(tooltip.querySelector('button')).not.toBeNull();
    expect(tooltip.textContent).not.toContain('Missing');
    expect(tooltip.textContent?.indexOf('Most')).toBeLessThan(
      tooltip.textContent.indexOf('Zero'),
    );
    expect(params.map((p) => p.seriesName)).toEqual([
      'Zero',
      'Missing',
      'Most',
    ]);
  });

  it.each([[], [{ value: null }], undefined])(
    'renders nothing when there are no observed values',
    (params) => {
      expect(tooltipFormatter(params)).toBe('');
    },
  );
});
