/**
 * TimelineHeader component tests.
 * Feature: 001-composable-timeline-bar
 */

import { act, cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TimelineHeader } from './TimelineHeader';

// Capture the onSelectionChange callback passed to TimeBrush so tests can verify forwarding
let capturedOnSelectionChange: ((start: number, end: number) => void) | undefined;

vi.mock('./TimeBrush', () => ({
  TimeBrush: ({
    onSelectionChange,
    children,
    className,
  }: {
    onSelectionChange?: (start: number, end: number) => void;
    children?: React.ReactNode;
    className?: string;
  }) => {
    capturedOnSelectionChange = onSelectionChange;

    return (
      <div className={className} data-testid="time-brush">
        <div className="cursor-ew-resize" />
        <div className="cursor-ew-resize" />
        {children}
      </div>
    );
  },
}));

afterEach(() => {
  cleanup();
});

describe('TimelineHeader', () => {
  const defaultProps = {
    minTime: new Date('2024-01-01T00:00:00Z'),
    maxTime: new Date('2024-01-01T00:00:10Z'), // 10 seconds total
    leftWidth: 40,
  };

  describe('time markers', () => {
    it('renders 5 time markers (0%, 25%, 50%, 75%, 100%)', () => {
      render(<TimelineHeader {...defaultProps} />);

      // Look for duration labels: 0ms (0%), 2.5s (25%), 5s (50%), 7.5s (75%), 10s (100%)
      expect(screen.getByText('0ms')).toBeTruthy(); // 0%
      expect(screen.getByText('2.5s')).toBeTruthy(); // 25%
      expect(screen.getByText('5s')).toBeTruthy(); // 50%
      expect(screen.getByText('7.5s')).toBeTruthy(); // 75%
      expect(screen.getByText('10s')).toBeTruthy(); // 100%
    });

    it('calculates correct durations for 1 minute timeline', () => {
      render(
        <TimelineHeader
          minTime={new Date('2024-01-01T00:00:00Z')}
          maxTime={new Date('2024-01-01T00:01:00Z')} // 60 seconds
          leftWidth={40}
        />
      );

      expect(screen.getByText('0ms')).toBeTruthy(); // 0%
      expect(screen.getByText('15s')).toBeTruthy(); // 25% of 60s
      expect(screen.getByText('30s')).toBeTruthy(); // 50% of 60s
      expect(screen.getByText('45s')).toBeTruthy(); // 75% of 60s
      expect(screen.getByText('1m')).toBeTruthy(); // 100% of 60s
    });

    it('handles millisecond-level timelines', () => {
      render(
        <TimelineHeader
          minTime={new Date('2024-01-01T00:00:00.000Z')}
          maxTime={new Date('2024-01-01T00:00:00.400Z')} // 400ms
          leftWidth={40}
        />
      );

      expect(screen.getByText('0ms')).toBeTruthy(); // 0%
      expect(screen.getByText('100ms')).toBeTruthy(); // 25% of 400ms
      expect(screen.getByText('200ms')).toBeTruthy(); // 50% of 400ms
      expect(screen.getByText('300ms')).toBeTruthy(); // 75% of 400ms
      expect(screen.getByText('400ms')).toBeTruthy(); // 100% of 400ms
    });
  });

  describe('layout', () => {
    it('applies leftWidth to the left spacer', () => {
      const { container } = render(<TimelineHeader {...defaultProps} leftWidth={35} />);

      const leftSpacer = container.querySelector('.shrink-0.pr-2') as HTMLElement;
      expect(leftSpacer.style.width).toBe('35%');
    });

    it('renders TimeBrush component', () => {
      const { container } = render(<TimelineHeader {...defaultProps} />);

      // TimeBrush renders handles with cursor-ew-resize
      const handles = container.querySelectorAll('.cursor-ew-resize');
      expect(handles.length).toBeGreaterThan(0);
    });

    it('renders the main timeline bar inside TimeBrush', () => {
      const { container } = render(<TimelineHeader {...defaultProps} />);

      // The main bar has bg-primary-moderate class
      expect(container.querySelector('.bg-primary-moderate')).toBeTruthy();
    });
  });

  describe('vertical guide lines', () => {
    it('renders 4 vertical guide lines (at 25%, 50%, 75%, 100%)', () => {
      const { container } = render(<TimelineHeader {...defaultProps} />);

      // Guide lines have bg-canvasSubtle and opacity-50 classes
      const guideLines = container.querySelectorAll('.bg-canvasSubtle.opacity-50');
      expect(guideLines).toHaveLength(4);
    });

    it('positions guide lines at correct percentages', () => {
      const { container } = render(<TimelineHeader {...defaultProps} />);

      const guideLines = container.querySelectorAll('.bg-canvasSubtle.opacity-50');
      const positions = Array.from(guideLines).map((line) => (line as HTMLElement).style.left);

      expect(positions).toContain('25%');
      expect(positions).toContain('50%');
      expect(positions).toContain('75%');
      expect(positions).toContain('100%');
    });
  });

  describe('selection callback', () => {
    it('forwards onSelectionChange from TimeBrush to parent', () => {
      const onSelectionChange = vi.fn();
      render(<TimelineHeader {...defaultProps} onSelectionChange={onSelectionChange} />);

      act(() => {
        capturedOnSelectionChange?.(30, 60);
      });

      expect(onSelectionChange).toHaveBeenCalledWith(30, 60);
    });

    it('does not crash without onSelectionChange', () => {
      expect(() => render(<TimelineHeader {...defaultProps} />)).not.toThrow();
    });
  });

  describe('split-color bar', () => {
    it('renders single full-width bar in default state', () => {
      const { container } = render(<TimelineHeader {...defaultProps} status="COMPLETED" />);

      // Default state: single bar element, no segments
      const defaultBar = container.querySelector('[data-testid="timeline-bar-default"]');
      expect(defaultBar).toBeTruthy();

      // No split segments should be present
      expect(container.querySelector('[data-testid="bar-segment-left"]')).toBeFalsy();
      expect(container.querySelector('[data-testid="bar-segment-middle"]')).toBeFalsy();
      expect(container.querySelector('[data-testid="bar-segment-right"]')).toBeFalsy();
    });

    it('renders 3 segments when selection is non-default', () => {
      const { container } = render(
        <TimelineHeader
          {...defaultProps}
          status="COMPLETED"
          selectionStart={25}
          selectionEnd={75}
        />
      );

      // Default bar should be gone
      expect(container.querySelector('[data-testid="timeline-bar-default"]')).toBeFalsy();

      // 3 segments should be present
      expect(container.querySelector('[data-testid="bar-segment-left"]')).toBeTruthy();
      expect(container.querySelector('[data-testid="bar-segment-middle"]')).toBeTruthy();
      expect(container.querySelector('[data-testid="bar-segment-right"]')).toBeTruthy();
    });

    it('outside segments use bg-canvasMuted', () => {
      const { container } = render(
        <TimelineHeader
          {...defaultProps}
          status="COMPLETED"
          selectionStart={25}
          selectionEnd={75}
        />
      );

      const left = container.querySelector('[data-testid="bar-segment-left"]') as HTMLElement;
      const right = container.querySelector('[data-testid="bar-segment-right"]') as HTMLElement;

      expect(left.className).toContain('bg-canvasMuted');
      expect(right.className).toContain('bg-canvasMuted');
    });

    it('middle segment uses status color class', () => {
      const { container } = render(
        <TimelineHeader
          {...defaultProps}
          status="COMPLETED"
          selectionStart={25}
          selectionEnd={75}
        />
      );

      const middle = container.querySelector('[data-testid="bar-segment-middle"]') as HTMLElement;
      expect(middle.className).toContain('bg-status-completed');
    });

    it('segments have correct widths based on selection', () => {
      const { container } = render(
        <TimelineHeader
          {...defaultProps}
          status="COMPLETED"
          selectionStart={25}
          selectionEnd={75}
        />
      );

      const left = container.querySelector('[data-testid="bar-segment-left"]') as HTMLElement;
      const middle = container.querySelector('[data-testid="bar-segment-middle"]') as HTMLElement;
      const right = container.querySelector('[data-testid="bar-segment-right"]') as HTMLElement;

      expect(left.style.width).toBe('25%');
      expect(middle.style.left).toBe('25%');
      expect(middle.style.width).toBe('50%');
      expect(right.style.left).toBe('75%');
      expect(right.style.width).toBe('25%');
    });

    it('forwards selection change to parent callback', () => {
      const onSelectionChange = vi.fn();
      render(<TimelineHeader {...defaultProps} onSelectionChange={onSelectionChange} />);

      act(() => {
        capturedOnSelectionChange?.(30, 60);
      });

      expect(onSelectionChange).toHaveBeenCalledWith(30, 60);
    });
  });

  describe('timestamp labels', () => {
    it('renders timestamp labels above handles when selection is non-default', () => {
      const { container } = render(
        <TimelineHeader {...defaultProps} selectionStart={25} selectionEnd={75} />
      );

      const leftLabel = container.querySelector('[data-testid="timestamp-label-left"]');
      const rightLabel = container.querySelector('[data-testid="timestamp-label-right"]');

      expect(leftLabel).toBeTruthy();
      expect(rightLabel).toBeTruthy();
    });

    it('hides timestamp labels when selection is at default position', () => {
      const { container } = render(<TimelineHeader {...defaultProps} />);

      // Default state (0, 100) — no labels
      expect(container.querySelector('[data-testid="timestamp-label-left"]')).toBeFalsy();
      expect(container.querySelector('[data-testid="timestamp-label-right"]')).toBeFalsy();
    });

    it('displays correct duration offsets', () => {
      // defaultProps: 10s total. Selection at 50%–100% → 5s and 10s
      const { container } = render(
        <TimelineHeader {...defaultProps} selectionStart={50} selectionEnd={100} />
      );

      const leftLabel = container.querySelector('[data-testid="timestamp-label-left"]');
      const rightLabel = container.querySelector('[data-testid="timestamp-label-right"]');

      expect(leftLabel?.textContent).toBe('5s');
      expect(rightLabel?.textContent).toBe('10s');
    });

    it('has z-20 class on timestamp labels', () => {
      const { container } = render(
        <TimelineHeader {...defaultProps} selectionStart={25} selectionEnd={75} />
      );

      const leftLabel = container.querySelector('[data-testid="timestamp-label-left"]');
      const rightLabel = container.querySelector('[data-testid="timestamp-label-right"]');

      expect((leftLabel as HTMLElement).className).toContain('z-20');
      expect((rightLabel as HTMLElement).className).toContain('z-20');
    });

    it('applies edge clamping near 0% (translateX(0))', () => {
      const { container } = render(
        <TimelineHeader {...defaultProps} selectionStart={2} selectionEnd={50} />
      );

      const leftLabel = container.querySelector(
        '[data-testid="timestamp-label-left"]'
      ) as HTMLElement;
      expect(leftLabel.style.transform).toBe('translateX(0)');
    });

    it('applies edge clamping near 100% (translateX(-100%))', () => {
      const { container } = render(
        <TimelineHeader {...defaultProps} selectionStart={50} selectionEnd={98} />
      );

      const rightLabel = container.querySelector(
        '[data-testid="timestamp-label-right"]'
      ) as HTMLElement;
      expect(rightLabel.style.transform).toBe('translateX(-100%)');
    });
  });

  describe('edge cases', () => {
    it('handles zero-duration timeline', () => {
      const sameTime = new Date('2024-01-01T00:00:00Z');

      // Should not throw
      expect(() =>
        render(<TimelineHeader minTime={sameTime} maxTime={sameTime} leftWidth={40} />)
      ).not.toThrow();
    });

    it('handles very long timeline (hours)', () => {
      render(
        <TimelineHeader
          minTime={new Date('2024-01-01T00:00:00Z')}
          maxTime={new Date('2024-01-01T02:00:00Z')} // 2 hours = 120 minutes
          leftWidth={40}
        />
      );

      // 2 hours = 7200000ms
      expect(screen.getByText('1h')).toBeTruthy(); // 50% of 2 hours = 1 hour
      expect(screen.getByText('2h')).toBeTruthy(); // 100% of 2 hours
    });
  });
});
