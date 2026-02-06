/**
 * TimeBrush component tests.
 */

import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TimeBrush } from './TimeBrush';

afterEach(() => {
  cleanup();
});

describe('TimeBrush', () => {
  describe('rendering', () => {
    it('renders without crashing', () => {
      render(<TimeBrush />);
      // Component should render (no specific test ID on container, so just verify no error)
      expect(document.body).toBeTruthy();
    });

    it('renders children inside the brush', () => {
      render(
        <TimeBrush>
          <div data-testid="child-content">Test content</div>
        </TimeBrush>
      );

      expect(screen.getByTestId('child-content')).toBeTruthy();
    });

    it('does not show reset button when at default selection', () => {
      render(<TimeBrush />);

      expect(screen.queryByTitle('Reset selection')).toBeNull();
    });

    it('applies custom className to container', () => {
      const { container } = render(<TimeBrush className="custom-class" />);

      expect((container.firstChild as HTMLElement).classList.contains('custom-class')).toBe(true);
    });
  });

  describe('selection callback', () => {
    it('calls onSelectionChange with initial values on mount', () => {
      const onSelectionChange = vi.fn();
      render(<TimeBrush onSelectionChange={onSelectionChange} />);

      expect(onSelectionChange).toHaveBeenCalledWith(0, 100);
    });

    it('calls onSelectionChange with custom initial values', () => {
      const onSelectionChange = vi.fn();
      render(<TimeBrush onSelectionChange={onSelectionChange} initialStart={25} initialEnd={75} />);

      expect(onSelectionChange).toHaveBeenCalledWith(25, 75);
    });
  });

  describe('reset button', () => {
    it('does not show reset button when at default selection', () => {
      render(<TimeBrush initialStart={0} initialEnd={100} showResetButton={true} />);

      // Initially at default, no reset button
      expect(screen.queryByTitle('Reset selection')).toBeNull();
    });

    it('does not show reset button when showResetButton is false', () => {
      render(<TimeBrush showResetButton={false} />);

      expect(screen.queryByTitle('Reset selection')).toBeNull();
    });

    // Note: Testing that reset button appears after drag requires mocking
    // getBoundingClientRect and simulating document-level mouse events.
    // This is better suited for integration/e2e tests.
  });

  describe('handle rendering', () => {
    it('renders left and right handles', () => {
      const { container } = render(<TimeBrush />);

      // Handles have cursor-ew-resize class
      const handles = container.querySelectorAll('.cursor-ew-resize');
      expect(handles).toHaveLength(2);
    });

    it('positions left handle at selectionStart', () => {
      const { container } = render(<TimeBrush initialStart={25} initialEnd={75} />);

      const handles = container.querySelectorAll('.cursor-ew-resize');
      const leftHandle = handles[0] as HTMLElement;

      expect(leftHandle.style.left).toBe('25%');
    });

    it('positions right handle at selectionEnd', () => {
      const { container } = render(<TimeBrush initialStart={25} initialEnd={75} />);

      const handles = container.querySelectorAll('.cursor-ew-resize');
      const rightHandle = handles[1] as HTMLElement;

      expect(rightHandle.style.left).toBe('75%');
    });
  });

  describe('selection area', () => {
    it('positions selection area between start and end', () => {
      const { container } = render(<TimeBrush initialStart={20} initialEnd={80} />);

      // Find the selection highlight div by its positioning style
      const selectionDivs = container.querySelectorAll('.absolute.top-0.h-full');
      // The selection div is the one with left/width styles (not handles)
      const selectionArea = Array.from(selectionDivs).find(
        (el) => (el as HTMLElement).style.left === '20%'
      ) as HTMLElement;

      expect(selectionArea).toBeTruthy();
      expect(selectionArea.style.left).toBe('20%');
      expect(selectionArea.style.width).toBe('60%'); // 80 - 20
    });
  });

  describe('default styling (FR-002, FR-003, FR-005)', () => {
    it('renders handles with visible gray color class (bg-surfaceMuted)', () => {
      const { container } = render(<TimeBrush />);

      // Handle inner divs should have bg-surfaceMuted class
      const handles = container.querySelectorAll('.bg-surfaceMuted');
      expect(handles).toHaveLength(2);
    });

    it('selection overlay has no visible styling by default', () => {
      const { container } = render(<TimeBrush />);

      // The old class should NOT be present
      expect(container.querySelector('.bg-primary-moderate\\/25')).toBeNull();
    });

    it('selection overlay div is still in the DOM for interaction', () => {
      const { container } = render(<TimeBrush initialStart={20} initialEnd={80} />);

      // Find the selection div by its positioning style
      const selectionDivs = container.querySelectorAll('.absolute.top-0.h-full');
      const selectionArea = Array.from(selectionDivs).find(
        (el) => (el as HTMLElement).style.left === '20%'
      ) as HTMLElement;

      expect(selectionArea).toBeTruthy();
      expect(selectionArea.style.width).toBe('60%');
    });

    it('reset button has system border styling', () => {
      const { container } = render(<TimeBrush />);

      // Mock getBoundingClientRect for the container
      const trackContainer = container.querySelector('.relative.h-4') as HTMLElement;
      trackContainer.getBoundingClientRect = vi.fn(() => ({
        left: 0,
        top: 0,
        right: 200,
        bottom: 16,
        width: 200,
        height: 16,
        x: 0,
        y: 0,
        toJSON: () => {},
      }));

      // Also mock on the outer container ref (used by the component)
      const outerContainer = container.firstChild as HTMLElement;
      outerContainer.getBoundingClientRect = vi.fn(() => ({
        left: 0,
        top: 0,
        right: 200,
        bottom: 16,
        width: 200,
        height: 16,
        x: 0,
        y: 0,
        toJSON: () => {},
      }));

      // Find the selection overlay (click target in default state) and click to create a selection
      const selectionOverlay = container.querySelector('.absolute.top-0.h-full') as HTMLElement;
      fireEvent.mouseDown(selectionOverlay, { clientX: 50 });
      fireEvent.mouseUp(document);

      const resetButton = container.querySelector('button[title="Reset selection"]') as HTMLElement;
      expect(resetButton).toBeTruthy();
      expect(resetButton.classList.contains('border')).toBe(true);
      expect(resetButton.classList.contains('border-muted')).toBe(true);
      expect(resetButton.classList.contains('bg-canvasBase')).toBe(true);
    });

    it('reset button does not have old background styling', () => {
      const { container } = render(<TimeBrush />);

      const outerContainer = container.firstChild as HTMLElement;
      outerContainer.getBoundingClientRect = vi.fn(() => ({
        left: 0,
        top: 0,
        right: 200,
        bottom: 16,
        width: 200,
        height: 16,
        x: 0,
        y: 0,
        toJSON: () => {},
      }));

      // Click to create a selection and trigger reset button to appear
      const selectionOverlay = container.querySelector('.absolute.top-0.h-full') as HTMLElement;
      fireEvent.mouseDown(selectionOverlay, { clientX: 50 });
      fireEvent.mouseUp(document);

      const resetButton = container.querySelector('button[title="Reset selection"]') as HTMLElement;
      expect(resetButton).toBeTruthy();
      expect(resetButton.classList.contains('bg-canvasSubtle')).toBe(false);
    });
  });

  describe('custom styling', () => {
    it('applies custom selectionClassName', () => {
      const { container } = render(<TimeBrush selectionClassName="custom-selection" />);

      expect(container.querySelector('.custom-selection')).toBeTruthy();
    });

    it('applies custom handleClassName', () => {
      const { container } = render(<TimeBrush handleClassName="custom-handle" />);

      const handles = container.querySelectorAll('.custom-handle');
      expect(handles).toHaveLength(2);
    });
  });

  describe('drag interactions', () => {
    // Note: Full drag interaction tests would require mocking getBoundingClientRect
    // and simulating document-level mouse events. These are integration-level tests.

    it('sets up mouse event listeners on mount', () => {
      const addEventListenerSpy = vi.spyOn(document, 'addEventListener');

      render(<TimeBrush />);

      expect(addEventListenerSpy).toHaveBeenCalledWith('mousemove', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('mouseup', expect.any(Function));

      addEventListenerSpy.mockRestore();
    });

    it('cleans up mouse event listeners on unmount', () => {
      const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener');

      const { unmount } = render(<TimeBrush />);
      unmount();

      expect(removeEventListenerSpy).toHaveBeenCalledWith('mousemove', expect.any(Function));
      expect(removeEventListenerSpy).toHaveBeenCalledWith('mouseup', expect.any(Function));

      removeEventListenerSpy.mockRestore();
    });

    it('left handle has mousedown handler', () => {
      const { container } = render(<TimeBrush />);

      const handles = container.querySelectorAll('.cursor-ew-resize');
      const leftHandle = handles[0] as HTMLElement;

      // Should not throw when clicking
      expect(() => fireEvent.mouseDown(leftHandle)).not.toThrow();
    });

    it('right handle has mousedown handler', () => {
      const { container } = render(<TimeBrush />);

      const handles = container.querySelectorAll('.cursor-ew-resize');
      const rightHandle = handles[1] as HTMLElement;

      // Should not throw when clicking
      expect(() => fireEvent.mouseDown(rightHandle)).not.toThrow();
    });
  });

  describe('minSelectionWidth', () => {
    it('accepts minSelectionWidth prop', () => {
      // This prop affects drag behavior, which is hard to test without full drag simulation
      // Just verify it renders without error
      expect(() => render(<TimeBrush minSelectionWidth={5} />)).not.toThrow();
    });
  });

  describe('re-selection behavior (Task 005)', () => {
    const mockRect = {
      left: 0,
      top: 0,
      right: 200,
      bottom: 16,
      width: 200,
      height: 16,
      x: 0,
      y: 0,
      toJSON: () => {},
    };

    function renderWithMock(props: Partial<Parameters<typeof TimeBrush>[0]> = {}) {
      const onSelectionChange = vi.fn();
      const result = render(<TimeBrush onSelectionChange={onSelectionChange} {...props} />);
      const outerContainer = result.container.firstChild as HTMLElement;
      outerContainer.getBoundingClientRect = vi.fn(() => ({ ...mockRect }));
      return { ...result, onSelectionChange, outerContainer };
    }

    /** Create a 25%-75% selection to enter non-default state */
    function makeNonDefaultSelection(outerContainer: HTMLElement) {
      const selectionOverlay = outerContainer.querySelector(
        '.absolute.top-0.h-full:not(.cursor-ew-resize)'
      ) as HTMLElement;
      fireEvent.mouseDown(selectionOverlay, { clientX: 50 }); // 25%
      fireEvent.mouseMove(document, { clientX: 150 }); // 75%
      fireEvent.mouseUp(document);
    }

    describe('click-and-drag re-selection', () => {
      it('clicking outside current selection creates a new selection', () => {
        const { outerContainer, onSelectionChange } = renderWithMock();
        makeNonDefaultSelection(outerContainer);
        onSelectionChange.mockClear();

        // Click on the background track outside selection (5% of 200px = 10px)
        const track = outerContainer.querySelector(
          '[data-testid="time-brush-track"]'
        ) as HTMLElement;
        fireEvent.mouseDown(track, { clientX: 10 });
        fireEvent.mouseMove(document, { clientX: 40 }); // drag to 20%
        fireEvent.mouseUp(document);

        expect(onSelectionChange).toHaveBeenCalledWith(5, 20);
      });

      it('clicking inside current selection preserves move behavior', () => {
        const { outerContainer, onSelectionChange } = renderWithMock();
        makeNonDefaultSelection(outerContainer);
        onSelectionChange.mockClear();

        // Click inside the selection overlay (50%, inside 25-75)
        const selectionOverlay = outerContainer.querySelector(
          '.absolute.top-0.h-full:not(.cursor-ew-resize)'
        ) as HTMLElement;
        fireEvent.mouseDown(selectionOverlay, { clientX: 100 });
        fireEvent.mouseMove(document, { clientX: 120 }); // drag right 10%
        fireEvent.mouseUp(document);

        expect(onSelectionChange).toHaveBeenCalledWith(35, 85);
      });
    });

    describe('hover line visibility', () => {
      it('shows hover line outside selection in non-default state', () => {
        const { outerContainer } = renderWithMock();
        makeNonDefaultSelection(outerContainer);

        // Hover on the track outside the selection (5%)
        const track = outerContainer.querySelector(
          '[data-testid="time-brush-track"]'
        ) as HTMLElement;
        fireEvent.mouseMove(track, { clientX: 10 });

        const cursorLine = outerContainer.querySelector('.pointer-events-none.w-px');
        expect(cursorLine).toBeTruthy();
      });

      it('does not show hover line inside selection in non-default state', () => {
        const { outerContainer } = renderWithMock();
        makeNonDefaultSelection(outerContainer);

        // Hover inside the selection overlay (50%, inside 25-75)
        const selectionOverlay = outerContainer.querySelector(
          '.absolute.top-0.h-full:not(.cursor-ew-resize)'
        ) as HTMLElement;
        fireEvent.mouseMove(selectionOverlay, { clientX: 100 });

        const cursorLine = outerContainer.querySelector('.pointer-events-none.w-px');
        expect(cursorLine).toBeNull();
      });

      it('shows hover line in default state (unchanged behavior)', () => {
        const { outerContainer } = renderWithMock();

        // Hover on the selection overlay in default state
        const selectionOverlay = outerContainer.querySelector(
          '.absolute.top-0.h-full:not(.cursor-ew-resize)'
        ) as HTMLElement;
        fireEvent.mouseMove(selectionOverlay, { clientX: 100 });

        const cursorLine = outerContainer.querySelector('.pointer-events-none.w-px');
        expect(cursorLine).toBeTruthy();
      });
    });

    describe('preserved interactions', () => {
      it('handle drag is preserved in non-default state', () => {
        const { outerContainer, onSelectionChange } = renderWithMock();
        makeNonDefaultSelection(outerContainer);
        onSelectionChange.mockClear();

        // Click on left handle and drag left
        const handles = outerContainer.querySelectorAll('.cursor-ew-resize');
        const leftHandle = handles[0] as HTMLElement;
        fireEvent.mouseDown(leftHandle, { clientX: 50 }); // at 25%
        fireEvent.mouseMove(document, { clientX: 30 }); // drag left 10%
        fireEvent.mouseUp(document);

        expect(onSelectionChange).toHaveBeenCalledWith(15, 75);
      });

      it('initial selection creation in default state still works', () => {
        const { outerContainer, onSelectionChange } = renderWithMock();
        onSelectionChange.mockClear();

        const selectionOverlay = outerContainer.querySelector(
          '.absolute.top-0.h-full:not(.cursor-ew-resize)'
        ) as HTMLElement;
        fireEvent.mouseDown(selectionOverlay, { clientX: 60 }); // 30%
        fireEvent.mouseMove(document, { clientX: 120 }); // 60%
        fireEvent.mouseUp(document);

        expect(onSelectionChange).toHaveBeenCalledWith(30, 60);
      });
    });
  });
});
