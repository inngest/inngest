import { useCallback, useEffect, useRef, useState } from 'react';

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../Tooltip/Tooltip';
import { cn } from '../utils/classNames';

const ELLIPSIS = '\u2026'; // Unicode ellipsis character

/** Ratio of available space given to the start vs end of the truncated string (40/60 favoring end) */
const START_RATIO = 0.4;

type TruncateMiddleProps = {
  /** The full text to display (truncated in the middle if it overflows) */
  text: string;
  /** Additional CSS classes for the outer span */
  className?: string;
};

/**
 * Measures pixel width of a string using an offscreen canvas context.
 * Caches the canvas across calls for performance.
 */
let _canvasCtx: CanvasRenderingContext2D | null = null;
let _canvasUnavailable = false;
function getCanvasContext(): CanvasRenderingContext2D | null {
  if (_canvasUnavailable) return null;
  if (!_canvasCtx) {
    if (typeof document === 'undefined') {
      _canvasUnavailable = true;
      return null;
    }
    try {
      const canvas = document.createElement('canvas');
      _canvasCtx = canvas.getContext('2d');
    } catch {
      // jsdom or SSR environment without canvas support
      _canvasUnavailable = true;
      return null;
    }
    if (!_canvasCtx) {
      _canvasUnavailable = true;
    }
  }
  return _canvasCtx;
}

function measureTextWidth(text: string, font: string): number {
  const ctx = getCanvasContext();
  if (!ctx) return 0;
  ctx.font = font;
  return ctx.measureText(text).width;
}

/**
 * Get the computed font shorthand string from an element,
 * matching what canvas.measureText needs.
 */
function getComputedFont(element: HTMLElement): string {
  const style = getComputedStyle(element);
  // Build the font shorthand: style weight size family
  return `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
}

/**
 * Compute a middle-truncated string that fits within `availableWidth` pixels.
 * Splits ~40% start / ~60% end around a single ellipsis character.
 *
 * Returns `null` if the full text already fits (no truncation needed).
 */
function computeMiddleTruncation(
  text: string,
  availableWidth: number,
  font: string
): string | null {
  const fullWidth = measureTextWidth(text, font);
  if (fullWidth <= availableWidth) return null;

  const ellipsisWidth = measureTextWidth(ELLIPSIS, font);
  const widthForText = availableWidth - ellipsisWidth;
  if (widthForText <= 0) return ELLIPSIS;

  const startBudget = widthForText * START_RATIO;
  const endBudget = widthForText * (1 - START_RATIO);

  // Walk from the start to find how many chars fit in startBudget
  let startChars = 0;
  let startWidth = 0;
  for (let i = 0; i < text.length; i++) {
    const charWidth = measureTextWidth(text[i]!, font);
    if (startWidth + charWidth > startBudget) break;
    startWidth += charWidth;
    startChars++;
  }

  // Walk from the end to find how many chars fit in endBudget
  let endChars = 0;
  let endWidth = 0;
  for (let i = text.length - 1; i >= startChars; i--) {
    const charWidth = measureTextWidth(text[i]!, font);
    if (endWidth + charWidth > endBudget) break;
    endWidth += charWidth;
    endChars++;
  }

  // Edge case: ensure we show at least 1 char on each side if possible
  if (startChars === 0 && text.length > 1) startChars = 1;
  if (endChars === 0 && text.length > 1) endChars = 1;

  const startPart = text.slice(0, startChars);
  const endPart = text.slice(text.length - endChars);

  return `${startPart}${ELLIPSIS}${endPart}`;
}

/**
 * TruncateMiddle - Renders text with middle truncation and a conditional tooltip.
 *
 * When the text fits its container, it renders normally.
 * When the text overflows, it truncates in the middle (keeping ~40% start, ~60% end)
 * and shows a tooltip with the full text on hover.
 */
export function TruncateMiddle({ text, className }: TruncateMiddleProps) {
  const containerRef = useRef<HTMLSpanElement>(null);
  const [displayText, setDisplayText] = useState(text);
  const [isTruncated, setIsTruncated] = useState(false);
  const [tooltipOpen, setTooltipOpen] = useState(false);

  const recalculate = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;

    const font = getComputedFont(el);
    const availableWidth = el.clientWidth;
    const truncated = computeMiddleTruncation(text, availableWidth, font);

    if (truncated) {
      setDisplayText(truncated);
      setIsTruncated(true);
    } else {
      setDisplayText(text);
      setIsTruncated(false);
      setTooltipOpen(false);
    }
  }, [text]);

  // Recalculate on mount and when text changes
  useEffect(() => {
    recalculate();
  }, [recalculate]);

  // Recalculate on container resize (e.g. user drags the left panel divider)
  useEffect(() => {
    const el = containerRef.current;
    if (!el || typeof ResizeObserver === 'undefined') return;

    const observer = new ResizeObserver(() => {
      recalculate();
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [recalculate]);

  const handleTooltipOpenChange = (open: boolean) => {
    if (!open) {
      setTooltipOpen(false);
      return;
    }
    // Only open tooltip if text is actually truncated
    if (isTruncated) {
      setTooltipOpen(true);
    }
  };

  return (
    <TooltipProvider>
      <Tooltip open={tooltipOpen} onOpenChange={handleTooltipOpenChange}>
        <TooltipTrigger asChild>
          <span
            ref={containerRef}
            className={cn('block overflow-hidden whitespace-nowrap', className)}
          >
            {displayText}
          </span>
        </TooltipTrigger>
        {isTruncated && (
          <TooltipContent
            side="top"
            className="flex min-h-8 max-w-md items-center break-all px-4 text-xs leading-[18px]"
          >
            {text}
          </TooltipContent>
        )}
      </Tooltip>
    </TooltipProvider>
  );
}
