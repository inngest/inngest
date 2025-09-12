import {
  RiCheckboxLine,
  RiCompassLine,
  RiFlashlightLine,
  RiRadarLine,
  RiRedPacketLine,
  RiRouteLine,
  RiSendPlane2Line,
  RiTerminalFill,
  RiTimerFlashLine,
  RiZzzLine,
} from '@remixicon/react';

import { Tooltip, TooltipContent, TooltipTrigger } from '../Tooltip';

type StepType = {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  color?: string;
};
const StepTypeIcon: Record<string, StepType> = {
  INVOKE: {
    icon: RiFlashlightLine,
    label: 'Invoke',
    color: 'text-quaternary-warmer-xIntense',
  },
  RUN: {
    icon: RiTerminalFill,
    label: 'Run',
    color: 'text-green-500',
  },
  SLEEP: {
    icon: RiZzzLine,
    label: 'Sleep',
    color: 'text-yellow-500',
  },
  WAIT_FOR_EVENT: {
    icon: RiTimerFlashLine,
    label: 'Wait for Event',
    color: 'text-purple-500',
  },
  AI_GATEWAY: {
    icon: RiRouteLine,
    label: 'AI Gateway',
    color: 'text-pink-500',
  },
  WAIT_FOR_SIGNAL: {
    icon: RiRadarLine,
    label: 'Wait for Signal',
    color: 'text-indigo-500',
  },
  'step.sendSignal': {
    icon: RiSendPlane2Line,
    label: 'Send Signal',
    color: 'text-orange-500',
  },
  'step.ai.infer': {
    icon: RiCompassLine,
    label: 'AI Infer',
    color: 'text-cyan-500',
  },
  'step.ai.wrap': {
    icon: RiRedPacketLine,
    label: 'AI Wrap',
    color: 'text-teal-500',
  },
  FINALIZATION: {
    icon: RiCheckboxLine,
    label: 'Finalization',
    color: 'text-teal-500',
  },
};
export const StepType = ({ stepType }: { stepType?: string | null | '' }) => {
  const Type = stepType ? StepTypeIcon[stepType] : null;
  return Type ? (
    <Tooltip>
      <TooltipTrigger>
        <div className="shrink-0 rounded-full border px-2 py-0.5">
          <Type.icon
            className={`left-[2.80px] top-[0.70px] mx-auto h-3.5 w-3.5 ${Type.color || ''}`}
          />
        </div>
      </TooltipTrigger>
      <TooltipContent className="whitespace-pre-line text-left">{Type.label}</TooltipContent>
    </Tooltip>
  ) : null;
};
