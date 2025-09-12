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

import { Pill } from '../Pill';

type StepType = {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  color?: string;
};
const StepTypeIcon: Record<string, StepType> = {
  INVOKE: {
    icon: RiFlashlightLine,
    label: 'Invoke',
  },
  RUN: {
    icon: RiTerminalFill,
    label: 'Run',
  },
  SLEEP: {
    icon: RiZzzLine,
    label: 'Sleep',
  },
  WAIT_FOR_EVENT: {
    icon: RiTimerFlashLine,
    label: 'Wait for Event',
  },
  AI_GATEWAY: {
    icon: RiRouteLine,
    label: 'AI Gateway',
  },
  WAIT_FOR_SIGNAL: {
    icon: RiRadarLine,
    label: 'Wait for Signal',
  },
  'step.sendSignal': {
    icon: RiSendPlane2Line,
    label: 'Send Signal',
  },
  'step.ai.infer': {
    icon: RiCompassLine,
    label: 'AI Infer',
  },
  'step.ai.wrap': {
    icon: RiRedPacketLine,
    label: 'AI Wrap',
  },
  FINALIZATION: {
    icon: RiCheckboxLine,
    label: 'Finalization',
  },
};
export const StepType = ({ stepType }: { stepType?: string | null | '' }) => {
  const TypePill = stepType ? StepTypeIcon[stepType] : null;
  return TypePill ? (
    <Pill appearance="outlined" kind="secondary">
      <TypePill.icon className="fill-quaternary-warmerxIntense h-2.5 w-2.5 shrink-0" />
    </Pill>
  ) : null;
};
