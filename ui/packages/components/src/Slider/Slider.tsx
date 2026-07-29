import { forwardRef } from 'react';
import * as SliderPrimitive from '@radix-ui/react-slider';

import { cn } from '../utils/classNames';

export const Slider = forwardRef<
  React.ElementRef<typeof SliderPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SliderPrimitive.Root> & {
    // Tints the filled range and the thumb border. Falls back to a neutral
    // contrast tone when omitted. Threaded through a CSS var so it overrides the
    // base class reliably (a second bg-*/border-* class would race on source
    // order).
    color?: string;
  }
>(({ className, color, style, ...props }, forwardedRef) => {
  return (
    <SliderPrimitive.Root
      {...props}
      ref={forwardedRef}
      style={color ? ({ ...style, '--slider-color': color } as React.CSSProperties) : style}
      className={cn(
        'relative flex w-full touch-none select-none items-center',
        props.disabled && 'pointer-events-none opacity-50',
        className
      )}
    >
      <SliderPrimitive.Track className="bg-surfaceMuted relative h-1 w-full grow overflow-hidden rounded-full">
        <SliderPrimitive.Range
          className={cn(
            'absolute h-full rounded-full',
            color ? 'bg-[var(--slider-color)]' : 'bg-contrast'
          )}
        />
      </SliderPrimitive.Track>
      <SliderPrimitive.Thumb
        className={cn(
          'bg-canvasBase block h-3.5 w-3.5 rounded-full border-2 shadow-sm outline-none transition-colors',
          color ? 'border-[var(--slider-color)]' : 'border-contrast'
        )}
      />
    </SliderPrimitive.Root>
  );
});

Slider.displayName = 'Slider';
