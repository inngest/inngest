import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiEyeLine, RiEyeOffLine } from '@remixicon/react';

type Props = {
  className?: string;
  isRevealed: boolean;
  onClick: () => void;
};

export function RevealButton({ className, isRevealed, onClick }: Props) {
  let Icon = RiEyeLine;
  let label = 'Reveal';
  if (isRevealed) {
    Icon = RiEyeOffLine;
    label = 'Hide';
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          aria-label={label}
          className={cn('flex items-center justify-center px-2', className)}
          onClick={onClick}
        >
          <Icon className="h-6" />
        </button>
      </TooltipTrigger>

      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
