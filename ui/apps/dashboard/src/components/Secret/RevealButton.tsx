import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiEyeLine, RiEyeOffLine } from '@remixicon/react';

type Props = {
  isRevealed: boolean;
  onClick: () => void;
};

export function RevealButton({ isRevealed, onClick }: Props) {
  let Icon = RiEyeLine;
  let label = 'Reveal';
  if (isRevealed) {
    Icon = RiEyeOffLine;
    label = 'Hide';
  }

  return (
    <Tooltip>
      <TooltipTrigger className="align-center flex">
        <button area-label={label} onClick={onClick}>
          <Icon className="h-6" />
        </button>
      </TooltipTrigger>

      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
