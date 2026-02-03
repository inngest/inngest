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

      // Find the selection highlight div (has the selection class)
      const selectionArea = container.querySelector('.bg-primary-moderate\\/25') as HTMLElement;

      expect(selectionArea.style.left).toBe('20%');
      expect(selectionArea.style.width).toBe('60%'); // 80 - 20
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
