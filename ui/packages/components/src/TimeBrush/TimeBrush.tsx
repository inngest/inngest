/**
 * TimeBrush component - Reusable range selection control.
 *
 * A brush-style UI for selecting a range within a container.
 * Features:
 * - Draggable left/right handles to resize selection
 * - Drag selection area to move the window
 * - Click and drag on track to create new selection
 * - Hover cursor line in default state
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
  selectionClassName = 'bg-primary-moderate/25',
  handleClassName = 'bg-primary-intense hover:bg-primary-xIntense',
  cursorLineClassName = 'bg-slate-500',
  children,
  className,
}: TimeBrushProps): JSX.Element {
  // Selection state (0-100 percentages)
  const [selectionStart, setSelectionStart] = useState(initialStart);
  const [selectionEnd, setSelectionEnd] = useState(initialEnd);

  // Hover position for cursor line (null when not hovering)
  const [hoverPosition, setHoverPosition] = useState<number | null>(null);

  // Drag state
  const dragModeRef = useRef<DragMode>('none');
  const dragStartXRef = useRef(0);
  const dragStartSelectionRef = useRef({ start: 0, end: 0 });
  const containerRef = useRef<HTMLDivElement>(null);

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

  // Handle mouse down on track to create a new selection (only for default state)
  const handleTrackMouseDown = useCallback(
    (e: React.MouseEvent) => {
      const container = containerRef.current;
      if (!container) return;

      // Only allow creating selection when in default state
      if (!isDefaultSelection) return;

      e.preventDefault();
      e.stopPropagation();

      const rect = container.getBoundingClientRect();
      const clickPercent = ((e.clientX - rect.left) / rect.width) * 100;

      // Set both start and end to the click position initially
      dragModeRef.current = 'create-selection';
      dragStartXRef.current = e.clientX;
      dragStartSelectionRef.current = { start: clickPercent, end: clickPercent };

      // Hide cursor line when dragging starts
      setHoverPosition(null);

      // Set initial selection at click point
      setSelectionStart(clickPercent);
      setSelectionEnd(clickPercent);
    },
    [isDefaultSelection]
  );

  // Handle mouse move on track to show cursor line (only for default state)
  const handleTrackMouseMove = useCallback(
    (e: React.MouseEvent) => {
      const container = containerRef.current;
      if (!container) return;

      // Only show cursor line in default state
      if (!isDefaultSelection) {
        setHoverPosition(null);
        return;
      }

      const rect = container.getBoundingClientRect();
      const hoverPercent = ((e.clientX - rect.left) / rect.width) * 100;
      setHoverPosition(Math.max(0, Math.min(100, hoverPercent)));
    },
    [isDefaultSelection]
  );

  // Handle mouse leave on track to hide cursor line
  const handleTrackMouseLeave = useCallback(() => {
    setHoverPosition(null);
  }, []);

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
        setSelectionStart(newStart);
      } else if (dragModeRef.current === 'right-handle') {
        // Move right handle, constrained by left handle
        const newEnd = Math.max(
          origStart + minSelectionWidth,
          Math.min(100, origEnd + deltaPercent)
        );
        setSelectionEnd(newEnd);
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

        setSelectionStart(newStart);
        setSelectionEnd(newEnd);
      } else if (dragModeRef.current === 'create-selection') {
        // Create selection by dragging from initial click point
        const currentPercent = ((e.clientX - rect.left) / rect.width) * 100;
        const clampedPercent = Math.max(0, Math.min(100, currentPercent));

        // origStart is the initial click position
        const clickPosition = origStart;

        if (clampedPercent < clickPosition) {
          // Dragging left from click point
          setSelectionStart(Math.max(0, clampedPercent));
          setSelectionEnd(clickPosition);
        } else {
          // Dragging right from click point
          setSelectionStart(clickPosition);
          setSelectionEnd(Math.min(100, clampedPercent));
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

  // Notify parent of selection changes
  useEffect(() => {
    onSelectionChange?.(selectionStart, selectionEnd);
  }, [selectionStart, selectionEnd, onSelectionChange]);

  // Reset selection to default
  const handleReset = useCallback(() => {
    setSelectionStart(initialStart);
    setSelectionEnd(initialEnd);
  }, [initialStart, initialEnd]);

  return (
    <div className={cn('relative', className)} ref={containerRef}>
      {/* Reset button */}
      {showResetButton && !isDefaultSelection && (
        <button
          onClick={handleReset}
          className="bg-canvasSubtle hover:bg-canvasMuted text-muted hover:text-basis absolute bottom-1 right-full mr-2 rounded px-2 py-0.5 text-xs transition-colors"
          title="Reset selection"
        >
          Reset
        </button>
      )}

      {/* Track container */}
      <div className="relative h-4">
        {/* Background track (clickable area for creating selection in default state) */}
        <div
          className={cn(
            'bg-canvasMuted absolute inset-0 top-1/2 h-1 -translate-y-1/2 rounded-full',
            isDefaultSelection ? 'cursor-default' : ''
          )}
          onMouseDown={isDefaultSelection ? handleTrackMouseDown : undefined}
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

        {/* Cursor line - shown when hovering in default (create selection) mode */}
        {isDefaultSelection && hoverPosition !== null && (
          <div
            className={cn('pointer-events-none absolute top-0 h-full w-px', cursorLineClassName)}
            style={{
              left: `${hoverPosition}%`,
              transform: 'translateX(-50%)',
              zIndex: 10,
            }}
          />
        )}

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
              'absolute left-1/2 top-0 h-full w-1 -translate-x-1/2 rounded-full transition-colors',
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
              'absolute left-1/2 top-0 h-full w-1 -translate-x-1/2 rounded-full transition-colors',
              handleClassName
            )}
          />
        </div>
      </div>
    </div>
  );
}
