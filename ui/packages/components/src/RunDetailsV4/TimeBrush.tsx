/**
 * TimeBrush component - Reusable range selection control.
 *
 * A brush-style UI for selecting a range within a container.
 * Features:
 * - Draggable left/right handles to resize selection
 * - Drag selection area to move the window
 * - Click and drag on track to create new selection (or re-select outside current)
 * - Hover cursor line in default state and outside selection
 * - Reset button when selection differs from default
 */

import { useCallback, useEffect, useRef, useState } from 'react';

import { cn } from '../utils/classNames';

export type TimeBrushProps = {
  /** Callback when selection changes (start and end as percentages 0-100) */
  onSelectionChange?: (start: number, end: number) => void;
  /** Initial selection start (default: 0) */
  initialStart?: number;
  /** Initial selection end (default: 100) */
  initialEnd?: number;
  /** Minimum width of selection in percentage (default: 2) */
  minSelectionWidth?: number;
  /** Whether to show the reset button (default: true) */
  showResetButton?: boolean;
  /** Custom class for the selection highlight */
  selectionClassName?: string;
  /** Custom class for the handles */
  handleClassName?: string;
  /** Custom class for the cursor line */
  cursorLineClassName?: string;
  /** Content to render inside the brush (e.g., a bar) */
  children?: React.ReactNode;
  /** Additional class name for the container */
  className?: string;
};

type DragMode = 'none' | 'left-handle' | 'right-handle' | 'selection' | 'create-selection';

/**
 * TimeBrush provides a brush-style range selection control.
 */
export function TimeBrush({
  onSelectionChange,
  initialStart = 0,
  initialEnd = 100,
  minSelectionWidth = 2,
  showResetButton = true,
  selectionClassName = '',
  handleClassName = 'bg-surfaceMuted hover:bg-muted',
  cursorLineClassName = 'bg-slate-500',
  children,
  className,
}: TimeBrushProps): JSX.Element {
  // Selection state (0-100 percentages) — single object for atomic updates from raw DOM listeners
  const [selection, setSelection] = useState({ start: initialStart, end: initialEnd });
  const { start: selectionStart, end: selectionEnd } = selection;

  // Cursor line — manipulated via ref + rAF to avoid re-renders on every mouse move
  const cursorLineRef = useRef<HTMLDivElement>(null);
  const rafIdRef = useRef<number>(0);

  // Drag state
  const dragModeRef = useRef<DragMode>('none');
  const dragStartXRef = useRef(0);
  const dragStartSelectionRef = useRef({ start: 0, end: 0 });
  const containerRef = useRef<HTMLDivElement>(null);
  const onSelectionChangeRef = useRef(onSelectionChange);
  onSelectionChangeRef.current = onSelectionChange;

  // Update cursor line position via rAF (bypasses React render cycle)
  const updateCursorLine = useCallback((position: number | null) => {
    cancelAnimationFrame(rafIdRef.current);
    rafIdRef.current = requestAnimationFrame(() => {
      const el = cursorLineRef.current;
      if (!el) return;
      if (position === null) {
        el.style.display = 'none';
      } else {
        el.style.display = '';
        el.style.left = `${position}%`;
      }
    });
  }, []);

  // Check if selection is at default (matches initial values)
  const isDefaultSelection = selectionStart === initialStart && selectionEnd === initialEnd;

  // Handle mouse down on left handle
  const handleLeftHandleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragModeRef.current = 'left-handle';
      dragStartXRef.current = e.clientX;
      dragStartSelectionRef.current = { start: selectionStart, end: selectionEnd };
    },
    [selectionStart, selectionEnd]
  );

  // Handle mouse down on right handle
  const handleRightHandleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragModeRef.current = 'right-handle';
      dragStartXRef.current = e.clientX;
      dragStartSelectionRef.current = { start: selectionStart, end: selectionEnd };
    },
    [selectionStart, selectionEnd]
  );

  // Handle mouse down on selection area (only for non-default state)
  const handleSelectionMouseDown = useCallback(
    (e: React.MouseEvent) => {
      // Only allow dragging selection when not in default state
      if (isDefaultSelection) return;

      e.preventDefault();
      e.stopPropagation();
      dragModeRef.current = 'selection';
      dragStartXRef.current = e.clientX;
      dragStartSelectionRef.current = { start: selectionStart, end: selectionEnd };
    },
    [selectionStart, selectionEnd, isDefaultSelection]
  );

  // Handle mouse down on track to create a new selection
  const handleTrackMouseDown = useCallback(
    (e: React.MouseEvent) => {
      const container = containerRef.current;
      if (!container) return;

      const rect = container.getBoundingClientRect();
      const clickPercent = ((e.clientX - rect.left) / rect.width) * 100;

      // Allow creating selection in default state OR outside current selection
      if (isDefaultSelection || clickPercent < selectionStart || clickPercent > selectionEnd) {
        e.preventDefault();
        e.stopPropagation();

        dragModeRef.current = 'create-selection';
        dragStartXRef.current = e.clientX;
        dragStartSelectionRef.current = { start: clickPercent, end: clickPercent };

        updateCursorLine(null);
      }
    },
    [isDefaultSelection, selectionStart, selectionEnd, updateCursorLine]
  );

  // Handle mouse move on track to show cursor line
  const handleTrackMouseMove = useCallback(
    (e: React.MouseEvent) => {
      const container = containerRef.current;
      if (!container) return;

      const rect = container.getBoundingClientRect();
      const hoverPercent = ((e.clientX - rect.left) / rect.width) * 100;
      const clampedPercent = Math.max(0, Math.min(100, hoverPercent));

      if (isDefaultSelection) {
        // Default state: show hover line everywhere
        updateCursorLine(clampedPercent);
      } else if (clampedPercent <= selectionStart || clampedPercent >= selectionEnd) {
        // Non-default: show hover line only outside selection
        updateCursorLine(clampedPercent);
      } else {
        // Non-default, inside selection: hide hover line
        updateCursorLine(null);
      }
    },
    [isDefaultSelection, selectionStart, selectionEnd, updateCursorLine]
  );

  // Handle mouse leave on track to hide cursor line
  const handleTrackMouseLeave = useCallback(() => {
    updateCursorLine(null);
  }, [updateCursorLine]);

  // Handle mouse move (global)
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (dragModeRef.current === 'none') return;

      const container = containerRef.current;
      if (!container) return;

      const rect = container.getBoundingClientRect();
      const deltaPixels = e.clientX - dragStartXRef.current;
      const deltaPercent = (deltaPixels / rect.width) * 100;

      const { start: origStart, end: origEnd } = dragStartSelectionRef.current;

      if (dragModeRef.current === 'left-handle') {
        // Move left handle, constrained by right handle
        const newStart = Math.max(
          0,
          Math.min(origEnd - minSelectionWidth, origStart + deltaPercent)
        );
        setSelection({ start: newStart, end: origEnd });
        onSelectionChangeRef.current?.(newStart, origEnd);
      } else if (dragModeRef.current === 'right-handle') {
        // Move right handle, constrained by left handle
        const newEnd = Math.max(
          origStart + minSelectionWidth,
          Math.min(100, origEnd + deltaPercent)
        );
        setSelection({ start: origStart, end: newEnd });
        onSelectionChangeRef.current?.(origStart, newEnd);
      } else if (dragModeRef.current === 'selection') {
        // Move entire selection, constrained by edges
        const width = origEnd - origStart;
        let newStart = origStart + deltaPercent;
        let newEnd = origEnd + deltaPercent;

        // Constrain to bounds
        if (newStart < 0) {
          newStart = 0;
          newEnd = width;
        }
        if (newEnd > 100) {
          newEnd = 100;
          newStart = 100 - width;
        }

        setSelection({ start: newStart, end: newEnd });
        onSelectionChangeRef.current?.(newStart, newEnd);
      } else if (dragModeRef.current === 'create-selection') {
        // Create selection by dragging from initial click point
        const currentPercent = ((e.clientX - rect.left) / rect.width) * 100;
        const clampedPercent = Math.max(0, Math.min(100, currentPercent));

        // origStart is the initial click position
        const clickPosition = origStart;

        const newStart =
          clampedPercent < clickPosition ? Math.max(0, clampedPercent) : clickPosition;
        const newEnd =
          clampedPercent < clickPosition ? clickPosition : Math.min(100, clampedPercent);

        // Only update when selection meets minimum width
        if (newEnd - newStart >= minSelectionWidth) {
          setSelection({ start: newStart, end: newEnd });
          onSelectionChangeRef.current?.(newStart, newEnd);
        }
      }
    };

    const handleMouseUp = () => {
      dragModeRef.current = 'none';
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [minSelectionWidth]);

  // Reset selection to default
  const handleReset = useCallback(() => {
    setSelection({ start: initialStart, end: initialEnd });
    onSelectionChangeRef.current?.(initialStart, initialEnd);
  }, [initialStart, initialEnd]);

  return (
    <div className={cn('relative', className)} ref={containerRef}>
      {/* Reset button */}
      {showResetButton && !isDefaultSelection && (
        <button
          onClick={handleReset}
          className="border-muted bg-canvasBase hover:bg-canvasSubtle text-basis absolute bottom-1 right-full mr-2 rounded border px-2 py-0.5 text-xs transition-colors"
          title="Reset selection"
        >
          Reset
        </button>
      )}

      {/* Track container */}
      <div className="relative h-4">
        {/* Background track (extended click target — overflows below the bar for easier interaction) */}
        <div
          data-testid="time-brush-track"
          className="absolute inset-0 -bottom-2 -top-6"
          onMouseDown={handleTrackMouseDown}
          onMouseMove={handleTrackMouseMove}
          onMouseLeave={handleTrackMouseLeave}
        />

        {/* Selection highlight area (full height) */}
        <div
          className={cn(
            'absolute top-0 h-full',
            selectionClassName,
            isDefaultSelection ? 'cursor-default' : 'cursor-grab active:cursor-grabbing'
          )}
          style={{
            left: `${selectionStart}%`,
            width: `${selectionEnd - selectionStart}%`,
          }}
          onMouseDown={isDefaultSelection ? handleTrackMouseDown : handleSelectionMouseDown}
          onMouseMove={handleTrackMouseMove}
          onMouseLeave={handleTrackMouseLeave}
        />

        {/* Children (e.g., the main bar) */}
        {children}

        {/* Cursor line - positioned via ref + rAF to avoid re-renders on mouse move */}
        <div
          ref={cursorLineRef}
          data-testid="cursor-line"
          className={cn('pointer-events-none absolute -top-6 bottom-0 w-px', cursorLineClassName)}
          style={{
            display: 'none',
            transform: 'translateX(-50%)',
            zIndex: 10,
          }}
        />

        {/* Left handle */}
        <div
          className="absolute top-0 h-full cursor-ew-resize"
          style={{
            left: `${selectionStart}%`,
            transform: 'translateX(-50%)',
            width: '16px',
          }}
          onMouseDown={handleLeftHandleMouseDown}
        >
          <div
            className={cn(
              'absolute left-1/2 top-0 h-full w-0.5 -translate-x-1/2 rounded-full transition-colors',
              handleClassName
            )}
          />
        </div>

        {/* Right handle */}
        <div
          className="absolute top-0 h-full cursor-ew-resize"
          style={{
            left: `${selectionEnd}%`,
            transform: 'translateX(-50%)',
            width: '16px',
          }}
          onMouseDown={handleRightHandleMouseDown}
        >
          <div
            className={cn(
              'absolute left-1/2 top-0 h-full w-0.5 -translate-x-1/2 rounded-full transition-colors',
              handleClassName
            )}
          />
        </div>
      </div>
    </div>
  );
}
