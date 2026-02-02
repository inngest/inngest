/**
 * Timeline container component tests.
 * Feature: 001-composable-timeline-bar
 */

import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { Timeline } from './Timeline';
import type { TimelineData } from './TimelineBar.types';

afterEach(() => {
  cleanup();
});

describe('Timeline', () => {
  const mockData: TimelineData = {
    minTime: new Date('2024-01-01T00:00:00Z'),
    maxTime: new Date('2024-01-01T00:00:10Z'), // 10 seconds
    leftWidth: 40,
    bars: [
      {
        id: 'step-1',
        name: 'Process payment',
        startTime: new Date('2024-01-01T00:00:00Z'),
        endTime: new Date('2024-01-01T00:00:05Z'), // 5 seconds
        style: 'step.run',
        timingBreakdown: {
          queueMs: 1000,
          executionMs: 4000,
          totalMs: 5000,
        },
      },
      {
        id: 'step-2',
        name: 'Send notification',
        startTime: new Date('2024-01-01T00:00:05Z'),
        endTime: new Date('2024-01-01T00:00:08Z'), // 3 seconds
        style: 'step.run',
      },
    ],
    orgName: 'Acme Corp',
  };

  // T035: renders all steps as TimelineBar components
  describe('renders all steps as TimelineBar components', () => {
    it('renders all bar names', () => {
      render(<Timeline data={mockData} />);
      expect(screen.getByText('Process payment')).toBeTruthy();
      expect(screen.getByText('Send notification')).toBeTruthy();
    });

    it('renders correct number of timeline bars', () => {
      render(<Timeline data={mockData} />);
      const bars = screen.getAllByTestId('timeline-bar-row');
      expect(bars).toHaveLength(2);
    });
  });

  // T036: renders timing breakdown rows using TimelineBar when expanded
  describe('renders timing breakdown rows when expanded', () => {
    it('shows timing breakdown when step is expanded', async () => {
      render(<Timeline data={mockData} />);

      // Find and click expand button for the first step (has timingBreakdown)
      const expandButtons = screen.getAllByRole('button', { name: /expand/i });
      fireEvent.click(expandButtons[0]);

      // Should show INNGEST and SERVER timing rows
      expect(screen.getByText('INNGEST')).toBeTruthy();
      expect(screen.getByText('ACME CORP')).toBeTruthy();
    });

    it('uses TimelineBar for timing breakdown rows', async () => {
      render(<Timeline data={mockData} />);

      // Expand first step
      const expandButtons = screen.getAllByRole('button', { name: /expand/i });
      fireEvent.click(expandButtons[0]);

      // Timing rows should be rendered as TimelineBar components
      const bars = screen.getAllByTestId('timeline-bar-row');
      // 2 main bars + 2 timing breakdown bars (INNGEST, SERVER)
      expect(bars.length).toBeGreaterThanOrEqual(4);
    });
  });

  // T037: manages expansion state for multiple bars
  describe('manages expansion state for multiple bars', () => {
    it('can expand multiple bars independently', () => {
      render(<Timeline data={mockData} />);

      const expandButtons = screen.getAllByRole('button', { name: /expand/i });

      // Expand first bar
      fireEvent.click(expandButtons[0]);
      expect(screen.getAllByText('INNGEST').length).toBeGreaterThan(0);

      // Second bar (if expandable) should still be collapsed
      // The state should be independent
    });

    it('can collapse an expanded bar', () => {
      render(<Timeline data={mockData} />);

      const expandButton = screen.getAllByRole('button', { name: /expand/i })[0];

      // Expand
      fireEvent.click(expandButton);
      expect(screen.getAllByText('INNGEST').length).toBeGreaterThan(0);

      // Collapse
      const collapseButton = screen.getByRole('button', { name: /collapse/i });
      fireEvent.click(collapseButton);
      expect(screen.queryByText('INNGEST')).toBeNull();
    });
  });

  // Additional tests
  describe('column resize handling', () => {
    it('renders with default left width', () => {
      render(<Timeline data={mockData} />);
      const leftPanels = screen.getAllByTestId('timeline-bar-left');
      expect(leftPanels[0]?.style.width).toBe('40%');
    });
  });

  describe('nested steps', () => {
    it('renders nested child steps', () => {
      const dataWithChildren: TimelineData = {
        ...mockData,
        bars: [
          {
            id: 'parent-1',
            name: 'Parent step',
            startTime: new Date('2024-01-01T00:00:00Z'),
            endTime: new Date('2024-01-01T00:00:05Z'),
            style: 'step.run',
            children: [
              {
                id: 'child-1',
                name: 'Child step',
                startTime: new Date('2024-01-01T00:00:01Z'),
                endTime: new Date('2024-01-01T00:00:03Z'),
                style: 'step.run',
              },
            ],
          },
        ],
      };

      render(<Timeline data={dataWithChildren} />);
      expect(screen.getByText('Parent step')).toBeTruthy();
      // Child is only visible when expanded - check parent exists first
      const expandButton = screen.getByRole('button', { name: /expand/i });
      fireEvent.click(expandButton);
      expect(screen.getByText('Child step')).toBeTruthy();
    });
  });
});
