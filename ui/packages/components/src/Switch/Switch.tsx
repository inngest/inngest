import { forwardRef, type HTMLAttributes } from 'react';
import * as SwitchPrimitive from '@radix-ui/react-switch';

import { cn } from '../utils/classNames';

export const Switch = forwardRef<
  React.ElementRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>
>(({ className, ...props }, forwardedRef) => {
  return (
    <SwitchPrimitive.Root
      {...props}
      ref={forwardedRef}
      className={`bg-surfaceMuted data-[state=checked]:bg-primary-moderate relative h-6 w-[42px] cursor-default rounded-full outline-none ${className}`}
    >
      <SwitchPrimitive.Thumb className="bg-alwaysWhite block h-5 w-5 translate-x-0.5 rounded-full transition-transform duration-100 will-change-transform data-[state=checked]:translate-x-[19px]" />
    </SwitchPrimitive.Root>
  );
});

export const SwitchWrapper = ({ children }: { children: React.ReactNode }) => (
  <div className="flex items-center gap-2">{children}</div>
);

interface SwitchLabelProps extends HTMLAttributes<HTMLLabelElement> {
  htmlFor: string;
}

export const SwitchLabel = ({ htmlFor, children, className }: SwitchLabelProps) => {
  return (
    <label htmlFor={htmlFor} className={cn('text-basis cursor-default font-medium', className)}>
      {children}
    </label>
  );
};
