import { RiArrowDownSLine, RiArrowRightSLine } from '@remixicon/react';

import { cn } from '../utils/classNames';

type Props = {
  className?: string;
  isExpanded: boolean;
  onClick: () => void;
};

export function ExpandButton({ className, isExpanded, onClick }: Props) {
  return (
    <button
      className={cn(
        'flex h-6 w-6 items-center justify-center rounded border border-slate-600',
        isExpanded ? 'bg-slate-600 text-slate-100' : 'bg-white text-slate-800',
        className
      )}
      onClick={onClick}
    >
      {isExpanded ? <RiArrowDownSLine className="h-4" /> : <RiArrowRightSLine className="h-4" />}
    </button>
  );
}
