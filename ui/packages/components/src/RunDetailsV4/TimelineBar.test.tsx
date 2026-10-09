/**
 * TimelineBar component tests.
 * Feature: 001-composable-timeline-bar
 *
 * Tests are written FIRST per TDD approach.
 */

import type { ReactNode } from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { TimelineBar } from './TimelineBar';

function Wrapper({ children }: { children: ReactNode }) {
  return <TooltipProvider>{children}</TooltipProvider>;
}

// jsdom doesn't provide ResizeObserver, which Radix tooltips use when open
beforeAll(() => {
  global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

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
      render(<TimelineBar {...defaultProps} />, { wrapper: Wrapper });
      expect(screen.getByText('Test Step')).toBeTruthy();
    });

    it('displays formatted duration', () => {
      render(<TimelineBar {...defaultProps} />, { wrapper: Wrapper });
      // 1234ms is >= 1s, so displayed in seconds with 3 decimal places
      expect(screen.getByText('1.234s')).toBeTruthy();
    });

    it('displays duration in milliseconds for short durations', () => {
      render(<TimelineBar {...defaultProps} duration={456} />, { wrapper: Wrapper });
      expect(screen.getByText('456ms')).toBeTruthy();
    });
  });

  // T010: positions bar using startPercent and widthPercent
  describe('positions bar using startPercent and widthPercent', () => {
    it('positions bar at correct left offset', () => {
      render(<TimelineBar {...defaultProps} startPercent={20} />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.left).toBe('20%');
    });

    it('sets bar width correctly', () => {
      render(<TimelineBar {...defaultProps} widthPercent={50} />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.width).toBe('50%');
    });
  });

  // T011: applies correct style based on style prop
  describe('applies correct style based on style prop', () => {
    it('applies step.run style colors (status-completed)', () => {
      render(<TimelineBar {...defaultProps} style="step.run" />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-status-completed');
    });

    it('applies timing.inngest style colors', () => {
      render(<TimelineBar {...defaultProps} style="timing.inngest" />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-surfaceMuted');
    });

    it('applies timing.server style colors (status-completed)', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-status-completed');
    });
  });

  // T012: renders with minimum 2px width for short durations (FR-009)
  describe('renders with minimum width for short durations', () => {
    it('applies minimum width for very small widthPercent', () => {
      render(<TimelineBar {...defaultProps} widthPercent={0.001} />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      // Should have min-width style applied
      expect(bar.style.minWidth).toBe('2px');
    });
  });

  // T020: renders expand toggle when expandable prop is true
  describe('expandable behavior', () => {
    it('renders expand toggle when expandable is true', () => {
      render(<TimelineBar {...defaultProps} expandable expanded={false} />, { wrapper: Wrapper });
      expect(screen.getByLabelText(/expand/i)).toBeTruthy();
    });

    it('does not render expand toggle when expandable is false', () => {
      render(<TimelineBar {...defaultProps} expandable={false} />, { wrapper: Wrapper });
      expect(screen.queryByLabelText(/expand/i)).toBeNull();
    });

    // T021: calls onToggle when row is clicked
    it('calls onToggle when row is clicked', () => {
      const onToggle = vi.fn();
      render(<TimelineBar {...defaultProps} expandable expanded={false} onToggle={onToggle} />, {
        wrapper: Wrapper,
      });
      fireEvent.click(screen.getByTestId('timeline-bar-row'));
      expect(onToggle).toHaveBeenCalledTimes(1);
    });

    // T022: renders children when expanded is true
    it('renders children when expanded is true', () => {
      render(
        <TimelineBar {...defaultProps} expandable expanded={true}>
          <div data-testid="child-content">Child content</div>
        </TimelineBar>,
        { wrapper: Wrapper }
      );
      expect(screen.getByTestId('child-content')).toBeTruthy();
    });

    // T023: hides children when expanded is false
    it('hides children when expanded is false', () => {
      render(
        <TimelineBar {...defaultProps} expandable expanded={false}>
          <div data-testid="child-content">Child content</div>
        </TimelineBar>,
        { wrapper: Wrapper }
      );
      expect(screen.queryByTestId('child-content')).toBeNull();
    });
  });

  // T028-T030: Visual styling tests (US3)
  describe('visual styling', () => {
    // T028: renders INNGEST timing with gray color
    it('renders INNGEST timing with correct styling', () => {
      render(<TimelineBar {...defaultProps} style="timing.inngest" />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-surfaceMuted');
    });

    // T029: renders SERVER timing with barber pole pattern (status-completed color)
    it('renders SERVER timing with barber pole pattern', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.className).toContain('bg-status-completed');
      // Should have background-image style for barber-pole
      expect(bar.style.backgroundImage).toContain('repeating-linear-gradient');
    });

    // T030: applies barber pole stripe pattern when configured
    it('applies barber pole pattern based on style configuration', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.backgroundImage).toBeTruthy();
    });
  });

  // Info icon with tooltip for Inngest and Your Server labels
  describe('info icon with tooltip', () => {
    it('renders info icon for Inngest timing bar', () => {
      render(<TimelineBar {...defaultProps} style="timing.inngest" />, { wrapper: Wrapper });
      const infoIcons = document.querySelectorAll('.cursor-help');
      expect(infoIcons.length).toBeGreaterThan(0);
    });

    it('renders info icon for Your Server timing bar', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />, { wrapper: Wrapper });
      const infoIcons = document.querySelectorAll('.cursor-help');
      expect(infoIcons.length).toBeGreaterThan(0);
    });

    it('does not render info icon for step.run style', () => {
      render(<TimelineBar {...defaultProps} style="step.run" />, { wrapper: Wrapper });
      const infoIcons = document.querySelectorAll('.cursor-help');
      expect(infoIcons.length).toBe(0);
    });
  });

  // T045-T046: Organization name tests (US5)
  describe('organization name in labels', () => {
    // T045: displays organization name in SERVER label when provided
    it('displays organization name in SERVER label', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" orgName="Acme Corp" />, {
        wrapper: Wrapper,
      });
      expect(screen.getByText('Acme Corp server')).toBeTruthy();
    });

    // T046: displays "Your server" when organization name not provided
    it('displays Your server when orgName not provided', () => {
      render(<TimelineBar {...defaultProps} style="timing.server" />, { wrapper: Wrapper });
      expect(screen.getByText('Your server')).toBeTruthy();
    });
  });

  // Depth/indentation tests
  describe('depth-based indentation', () => {
    it('applies indentation based on depth', () => {
      render(<TimelineBar {...defaultProps} depth={2} />, { wrapper: Wrapper });
      const leftPanel = screen.getByTestId('timeline-bar-left');
      // Should have padding-left based on BASE_LEFT_PADDING_PX + depth * INDENT_WIDTH_PX
      // 4px base + 2 * 20px = 44px
      expect(leftPanel.style.paddingLeft).toBe('44px');
    });

    it('has base indentation at depth 0', () => {
      render(<TimelineBar {...defaultProps} depth={0} />, { wrapper: Wrapper });
      const leftPanel = screen.getByTestId('timeline-bar-left');
      // BASE_LEFT_PADDING_PX = 4px
      expect(leftPanel.style.paddingLeft).toBe('4px');
    });
  });

  // Expanded row opacity (EXE-1217, Task 001)
  describe('expanded row opacity', () => {
    it('renders expanded parent row with opacity 0', () => {
      render(<TimelineBar {...defaultProps} expandable expanded={true} />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.opacity).toBe('0');
    });

    it('renders collapsed row with full opacity', () => {
      render(<TimelineBar {...defaultProps} expandable expanded={false} />, { wrapper: Wrapper });
      const bar = screen.getByTestId('timeline-bar-visual');
      expect(bar.style.opacity).toBe('1');
    });
  });

  // Hover tooltip tests
  describe('hover tooltip', () => {
    const tooltipProps = {
      ...defaultProps,
      startTime: new Date('2024-01-15T10:00:00.500Z'),
      endTime: new Date('2024-01-15T10:00:01.200Z'),
      minTime: new Date('2024-01-15T10:00:00.000Z'),
    };

    it('renders tooltip trigger on right panel when time props provided', () => {
      render(<TimelineBar {...tooltipProps} />, { wrapper: Wrapper });
      expect(screen.getByTestId('timeline-bar-right')).toBeTruthy();
    });

    it('renders right panel without tooltip when time props are missing', () => {
      render(<TimelineBar {...defaultProps} />, { wrapper: Wrapper });
      expect(screen.getByTestId('timeline-bar-right')).toBeTruthy();
    });

    it('does not crash when endTime is null', () => {
      render(<TimelineBar {...tooltipProps} endTime={null} />, { wrapper: Wrapper });
      expect(screen.getByTestId('timeline-bar-right')).toBeTruthy();
    });
  });

  // Selection tests
  describe('selection', () => {
    it('applies selected styling when selected is true', () => {
      render(<TimelineBar {...defaultProps} selected />, { wrapper: Wrapper });
      const row = screen.getByTestId('timeline-bar-row');
      const highlight = row.querySelector('.bg-secondary-3xSubtle');
      expect(highlight).toBeTruthy();
    });

    it('calls onClick when row is clicked', () => {
      const onClick = vi.fn();
      render(<TimelineBar {...defaultProps} onClick={onClick} />, { wrapper: Wrapper });
      fireEvent.click(screen.getByTestId('timeline-bar-row'));
      expect(onClick).toHaveBeenCalledTimes(1);
    });
  });

  describe('score badge', () => {
    it('renders no badge when the span has no scores', () => {
      render(<TimelineBar {...defaultProps} />, { wrapper: Wrapper });
      expect(screen.queryByTestId('score-badge')).toBeNull();
    });

    it('renders a badge when the span has scores', () => {
      render(<TimelineBar {...defaultProps} scores={[{ name: 'relevance', value: 0.9 }]} />, {
        wrapper: Wrapper,
      });
      expect(screen.getByTestId('score-badge')).toBeTruthy();
    });

    it('lets badge clicks fall through to row selection', () => {
      const onClick = vi.fn();
      render(
        <TimelineBar
          {...defaultProps}
          onClick={onClick}
          scores={[{ name: 'relevance', value: 0.9 }]}
        />,
        { wrapper: Wrapper }
      );
      fireEvent.click(screen.getByTestId('score-badge'));
      expect(onClick).toHaveBeenCalledTimes(1);
    });
  });
});

describe('TimelineBar warning icon', () => {
  const defaultProps = {
    name: 'Test Step',
    duration: 1234,
    startPercent: 10,
    widthPercent: 25,
    depth: 0,
    leftWidth: 40,
    style: 'step.run' as const,
  };

  it('renders no icon without warnings', () => {
    render(<TimelineBar {...defaultProps} />, { wrapper: Wrapper });
    expect(screen.queryByTestId('warning-icon')).toBeNull();
  });

  it('renders an accessible icon carrying the message', () => {
    render(
      <TimelineBar {...defaultProps} warnings={[{ key: 'k', message: 'build it earlier' }]} />,
      {
        wrapper: Wrapper,
      }
    );
    expect(screen.getByTestId('warning-icon').getAttribute('aria-label')).toBe(
      'Warning: build it earlier'
    );
  });

  it('summarizes multiple warnings as a count', () => {
    render(
      <TimelineBar
        {...defaultProps}
        warnings={[
          { key: 'a', message: 'x' },
          { key: 'b', message: 'y' },
        ]}
      />,
      { wrapper: Wrapper }
    );
    expect(screen.getByTestId('warning-icon').getAttribute('aria-label')).toBe(
      'Warning: 2 warnings'
    );
  });
});

describe('TimelineBar warning icon placement and access', () => {
  const props = {
    name: 'A very long step name that would be truncated with an ellipsis',
    duration: 1234,
    startPercent: 10,
    widthPercent: 25,
    depth: 0,
    leftWidth: 40,
    style: 'step.run' as const,
    warnings: [{ key: 'k', message: 'build it earlier' }],
  };

  it('is not inside the truncating name span', () => {
    render(<TimelineBar {...props} />, { wrapper: Wrapper });
    const icon = screen.getByTestId('warning-icon');
    expect(icon.closest('.text-ellipsis')).toBeNull();
  });

  it('is focusable and shows its tooltip on focus', async () => {
    render(<TimelineBar {...props} />, { wrapper: Wrapper });
    const icon = screen.getByTestId('warning-icon');
    expect(icon.getAttribute('tabindex')).toBe('0');
    expect(icon.getAttribute('aria-label')).toBe('Warning: build it earlier');

    fireEvent.focus(icon);
    expect((await screen.findAllByText('build it earlier')).length).toBeGreaterThan(0);
  });

  it('selects the row when clicked', () => {
    const onClick = vi.fn();
    render(<TimelineBar {...props} onClick={onClick} />, { wrapper: Wrapper });

    fireEvent.click(screen.getByTestId('warning-icon'));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('lists every message in its tooltip', async () => {
    render(
      <TimelineBar
        {...props}
        warnings={[
          { key: 'a', message: 'first warning' },
          { key: 'b', message: 'second warning' },
        ]}
      />,
      { wrapper: Wrapper }
    );

    fireEvent.focus(screen.getByTestId('warning-icon'));

    expect((await screen.findAllByText('first warning')).length).toBeGreaterThan(0);
    expect((await screen.findAllByText('second warning')).length).toBeGreaterThan(0);
  });
});

describe('TimelineBar dimmed', () => {
  const props = {
    name: 'create sandbox',
    duration: 1234,
    startPercent: 10,
    widthPercent: 25,
    depth: 0,
    leftWidth: 40,
    style: 'step.run' as const,
  };

  it('fades the name, duration and bar', () => {
    render(<TimelineBar {...props} dimmed />, { wrapper: Wrapper });

    expect(screen.getByText('create sandbox').className).toContain('text-light');
    expect(screen.getByText('1.234s').className).toContain('text-light');
    expect(screen.getByTestId('timeline-bar-track').className).toContain('opacity-50');
  });

  it('draws a row at full strength by default', () => {
    render(<TimelineBar {...props} />, { wrapper: Wrapper });

    expect(screen.getByText('create sandbox').className).not.toContain('text-light');
    expect(screen.getByTestId('timeline-bar-track').className).not.toContain('opacity-50');
  });
});
