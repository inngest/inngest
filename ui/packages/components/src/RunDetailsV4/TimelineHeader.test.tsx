/**
 * TimelineHeader component tests.
 * Feature: 001-composable-timeline-bar
 */

import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TimelineHeader } from './TimelineHeader';

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

      // Look for duration labels: 0ms (0%), 2.50s (25%), 5s (50%), 7.50s (75%), 10s (100%)
      expect(screen.getByText('<1ms')).toBeTruthy(); // 0%
      expect(screen.getByText('2.50s')).toBeTruthy(); // 25%
      expect(screen.getByText('5.00s')).toBeTruthy(); // 50%
      expect(screen.getByText('7.50s')).toBeTruthy(); // 75%
      expect(screen.getByText('10.00s')).toBeTruthy(); // 100%
    });

    it('calculates correct durations for 1 minute timeline', () => {
      render(
        <TimelineHeader
          minTime={new Date('2024-01-01T00:00:00Z')}
          maxTime={new Date('2024-01-01T00:01:00Z')} // 60 seconds
          leftWidth={40}
        />
      );

      expect(screen.getByText('<1ms')).toBeTruthy(); // 0%
      expect(screen.getByText('15.00s')).toBeTruthy(); // 25% of 60s
      expect(screen.getByText('30.00s')).toBeTruthy(); // 50% of 60s
      expect(screen.getByText('45.00s')).toBeTruthy(); // 75% of 60s
      expect(screen.getByText('1.00m')).toBeTruthy(); // 100% of 60s
    });

    it('handles millisecond-level timelines', () => {
      render(
        <TimelineHeader
          minTime={new Date('2024-01-01T00:00:00.000Z')}
          maxTime={new Date('2024-01-01T00:00:00.400Z')} // 400ms
          leftWidth={40}
        />
      );

      expect(screen.getByText('<1ms')).toBeTruthy(); // 0%
      expect(screen.getByText('100.00ms')).toBeTruthy(); // 25% of 400ms
      expect(screen.getByText('200.00ms')).toBeTruthy(); // 50% of 400ms
      expect(screen.getByText('300.00ms')).toBeTruthy(); // 75% of 400ms
      expect(screen.getByText('400.00ms')).toBeTruthy(); // 100% of 400ms
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
    it('passes onSelectionChange to TimeBrush', () => {
      const onSelectionChange = vi.fn();
      render(<TimelineHeader {...defaultProps} onSelectionChange={onSelectionChange} />);

      // TimeBrush calls onSelectionChange on mount with default values
      expect(onSelectionChange).toHaveBeenCalledWith(0, 100);
    });

    it('does not crash without onSelectionChange', () => {
      expect(() => render(<TimelineHeader {...defaultProps} />)).not.toThrow();
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

      // 2 hours = 7200000ms, formatted as minutes
      expect(screen.getByText('60.00m')).toBeTruthy(); // 50% of 2 hours = 60 minutes
      expect(screen.getByText('120.00m')).toBeTruthy(); // 100% of 2 hours = 120 minutes
    });
  });
});
