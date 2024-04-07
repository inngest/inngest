import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { IconEye } from '@inngest/components/icons/Eye';
import { IconEyeSlash } from '@inngest/components/icons/EyeSlash';

type Props = {
  isRevealed: boolean;
  onClick: () => void;
};

export function RevealButton({ isRevealed, onClick }: Props) {
  let Icon = IconEye;
  let label = 'Reveal';
  if (isRevealed) {
    Icon = IconEyeSlash;
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
