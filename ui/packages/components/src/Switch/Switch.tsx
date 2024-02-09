import { forwardRef } from 'react';
import * as SwitchPrimitive from '@radix-ui/react-switch';

export const Switch = forwardRef<
  React.ElementRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>
>(({ ...props }, forwardedRef) => {
  return (
    <SwitchPrimitive.Root
      {...props}
      ref={forwardedRef}
      className="relative h-6 w-[42px] cursor-default rounded-full bg-slate-600 outline-none data-[state=checked]:bg-indigo-600"
    >
      <SwitchPrimitive.Thumb className="block h-5 w-5 translate-x-0.5 rounded-full bg-white transition-transform duration-100 will-change-transform data-[state=checked]:translate-x-[19px]" />
    </SwitchPrimitive.Root>
  );
});

export const SwitchWrapper = ({ children }: { children: React.ReactNode }) => (
  <div className="flex items-center gap-2">{children}</div>
);

export const SwitchLabel = forwardRef<HTMLLabelElement, { children: string }>(
  ({ children }, forwardedRef) => (
    <label ref={forwardedRef} className="font-medium text-slate-900">
      {children}
    </label>
  )
);
