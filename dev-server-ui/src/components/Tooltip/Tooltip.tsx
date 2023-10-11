import React from 'react';
import * as TooltipPrimitive from '@radix-ui/react-tooltip';

export default function Tooltip({ children, content, ...props }) {
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
            className="animate-slide-down-fade bg-slate-400 rounded-lg px-2 py-1 text-sm text-slate-800 z-50"
          >
            {content}
            <TooltipPrimitive.Arrow width={11} height={5} className="fill-slate-400" />
          </TooltipPrimitive.Content>
        </TooltipPrimitive.Portal>
      </TooltipPrimitive.Root>
    </TooltipPrimitive.Provider>
  );
}
