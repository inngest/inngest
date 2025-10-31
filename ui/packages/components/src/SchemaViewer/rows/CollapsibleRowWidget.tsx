'use client';

import { RiArrowDownSFill } from '@remixicon/react';

export type CollapsibleRowWidgetProps = { open: boolean };

export function CollapsibleRowWidget({ open }: CollapsibleRowWidgetProps) {
  return (
    <span className="text-muted -mb-0.5 inline-flex h-4 w-4 items-center justify-center">
      <RiArrowDownSFill
        className={`text-light h-4 w-4 transition-transform ${open ? 'rotate-0' : '-rotate-90'}`}
      />
    </span>
  );
}
