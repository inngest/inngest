import React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';

type TooltipProps = {
  children: React.ReactNode;
  content: React.ReactNode;
};

export function Tooltip({ children, content, ...props }: TooltipProps) {
  return (
    <TooltipPrimitive.Provider>
      <TooltipPrimitive.Root>
        <TooltipPrimitive.Trigger>{children}</TooltipPrimitive.Trigger>
        <TooltipPrimitive.Portal>
          <TooltipPrimitive.Content
            side="top"
            align="center"
            {...props}
            sideOffset={1}
            className="animate-slide-down-fade z-50 rounded-lg bg-slate-400 px-2 py-1 text-sm text-slate-800"
          >
            {content}
            <TooltipPrimitive.Arrow width={11} height={5} className="fill-slate-400" />
          </TooltipPrimitive.Content>
        </TooltipPrimitive.Portal>
      </TooltipPrimitive.Root>
    </TooltipPrimitive.Provider>
  );
}
