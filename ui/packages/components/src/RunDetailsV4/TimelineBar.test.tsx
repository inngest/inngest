/**
 * TimelineBar component tests.
 * Feature: 001-composable-timeline-bar
 *
 * Tests are written FIRST per TDD approach.
 */

import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TimelineBar } from './TimelineBar';

afterEach(() => {
  cleanup();
});

describe('TimelineBar', () => {
  const defaultProps = {
    name: 'Test Step',
    duration: 1234,
    startPercent: 10,
    widthPercent: 25,
    depth: 0,
    leftWidth: 40,
    style: 'step.run' as const,
  };

  // T009: renders bar with name and duration
  describe('renders bar with name and duration', () => {
    it('displays the step name', () => {
      render(<TimelineBar {...defaultProps} />);
      expect(screen.getByText('Test Step')).toBeTruthy();
    });

    it('displays formatted duration', () => {
      render(<TimelineBar {...defaultProps} />);
      // 1234ms should format to "1.2s"
      expect(screen.getByText('1.2s')).toBeTruthy();
    });

    it('displays duration in milliseconds for short durations', () => {
      render(<TimelineBar {...defaultProps} duration={456} />);
      expect(screen.getByText('456ms')).toBeTruthy();
    });
  });

  // T010: positions bar using startPercent and widthPercent
  describe('positions bar using startPercent and widthPercent', () => {
    it('positions bar at correct left offset', () => {
      render(<TimelineBar {...defaultProps} startPercent={20} />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.left).toBe('20%');
    });

    it('sets bar width correctly', () => {
      render(<TimelineBar {...defaultProps} widthPercent={50} />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.width).toBe('50%');
    });
  });

  // T011: applies correct style based on style prop
  describe('applies correct style based on style prop', () => {
    it('applies step.run style colors (status-completed)', () => {
      render(<TimelineBar {...defaultProps} style="step.run" />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-status-completed');
    });

    it('applies timing.inngest style colors', () => {
      render(<TimelineBar {...defaultProps} style="timing.inngest" />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-slate-300');
    });

    it('applies timing.server style colors (status-completed)', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-status-completed');
    });
  });

  // T012: renders with minimum 2px width for short durations (FR-009)
  describe('renders with minimum width for short durations', () => {
    it('applies minimum width for very small widthPercent', () => {
      render(<TimelineBar {...defaultProps} widthPercent={0.001} />);
      const bar = screen.getByTestId('timeline-bar-visual');
      // Should have min-width style applied
      expect(bar.style.minWidth).toBe('2px');
    });
  });

  // T020: renders expand toggle when expandable prop is true
  describe('expandable behavior', () => {
    it('renders expand toggle when expandable is true', () => {
      render(<TimelineBar {...defaultProps} expandable expanded={false} />);
      expect(screen.getByRole('button', { name: /expand/i })).toBeTruthy();
    });

    it('does not render expand toggle when expandable is false', () => {
      render(<TimelineBar {...defaultProps} expandable={false} />);
      expect(screen.queryByRole('button', { name: /expand/i })).toBeNull();
    });

    // T021: calls onToggle when expand button clicked
    it('calls onToggle when expand button is clicked', () => {
      const onToggle = vi.fn();
      render(<TimelineBar {...defaultProps} expandable expanded={false} onToggle={onToggle} />);
      fireEvent.click(screen.getByRole('button', { name: /expand/i }));
      expect(onToggle).toHaveBeenCalledTimes(1);
    });

    // T022: renders children when expanded is true
    it('renders children when expanded is true', () => {
      render(
        <TimelineBar {...defaultProps} expandable expanded={true}>
          <div data-testid="child-content">Child content</div>
        </TimelineBar>
      );
      expect(screen.getByTestId('child-content')).toBeTruthy();
    });

    // T023: hides children when expanded is false
    it('hides children when expanded is false', () => {
      render(
        <TimelineBar {...defaultProps} expandable expanded={false}>
          <div data-testid="child-content">Child content</div>
        </TimelineBar>
      );
      expect(screen.queryByTestId('child-content')).toBeNull();
    });
  });

  // T028-T030: Visual styling tests (US3)
  describe('visual styling', () => {
    // T028: renders INNGEST timing with gray color
    it('renders INNGEST timing with correct styling', () => {
      render(<TimelineBar {...defaultProps} style="timing.inngest" />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-slate-300');
    });

    // T029: renders SERVER timing with barber pole pattern (status-completed color)
    it('renders SERVER timing with barber pole pattern', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-status-completed');
      // Should have background-image style for barber-pole
      expect(bar.style.backgroundImage).toContain('repeating-linear-gradient');
    });

    // T030: applies barber pole stripe pattern when configured
    it('applies barber pole pattern based on style configuration', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />);
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.backgroundImage).toBeTruthy();
    });
  });

  // T045-T046: Organization name tests (US5)
  describe('organization name in labels', () => {
    // T045: displays organization name in SERVER label when provided
    it('displays organization name in SERVER label', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" orgName="Acme Corp" />);
      expect(screen.getByText('ACME CORP')).toBeTruthy();
    });

    // T046: displays "YOUR SERVER" when organization name not provided
    it('displays YOUR SERVER when orgName not provided', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />);
      expect(screen.getByText('YOUR SERVER')).toBeTruthy();
    });
  });

  // Depth/indentation tests
  describe('depth-based indentation', () => {
    it('applies indentation based on depth', () => {
      render(<TimelineBar {...defaultProps} depth={2} />);
      const leftPanel = screen.getByTestId('timeline-bar-left');
      // Should have padding-left based on BASE_LEFT_PADDING_PX + depth * INDENT_WIDTH_PX
      // 4px base + 2 * 20px = 44px
      expect(leftPanel.style.paddingLeft).toBe('44px');
    });

    it('has base indentation at depth 0', () => {
      render(<TimelineBar {...defaultProps} depth={0} />);
      const leftPanel = screen.getByTestId('timeline-bar-left');
      // BASE_LEFT_PADDING_PX = 4px
      expect(leftPanel.style.paddingLeft).toBe('4px');
    });
  });

  // Selection tests
  describe('selection', () => {
    it('applies selected styling when selected is true', () => {
      render(<TimelineBar {...defaultProps} selected />);
      const row = screen.getByTestId('timeline-bar-row');
      expect(row.className).toContain('bg-canvasSubtle');
    });

    it('calls onClick when row is clicked', () => {
      const onClick = vi.fn();
      render(<TimelineBar {...defaultProps} onClick={onClick} />);
      fireEvent.click(screen.getByTestId('timeline-bar-row'));
      expect(onClick).toHaveBeenCalledTimes(1);
    });
  });
});
