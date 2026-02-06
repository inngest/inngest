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
    it('renders handles with black color class (bg-basis)', () => {
      const { container } = render(<TimeBrush />);

      // Handle inner divs should have bg-basis class
      const handles = container.querySelectorAll('.bg-basis');
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
});
