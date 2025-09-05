import {
  RiArrowRightLine,
  RiCpuFill,
  RiMailLine,
  RiPauseCircleLine,
  RiPlayFill,
  RiRadioButtonLine,
  RiRobot2Fill,
  RiSendPlaneFill,
} from '@remixicon/react';

import { Pill } from '../Pill';

type Props = {
  stepType: string;
};

// XXX: these icons are all just Claude's suggestions and need to be corrected based
// on Figma
export function StepTypePill({ stepType }: Props) {
  const getStepConfig = (type: string) => {
    const upperType = type.toUpperCase();

    switch (upperType) {
      case 'INVOKE':
        return {
          icon: <RiArrowRightLine className="h-3 w-3" />,
          kind: 'primary' as const,
          text: 'INVOKE',
        };
      case 'RUN':
        return {
          icon: <RiPlayFill className="h-3 w-3" />,
          kind: 'info' as const,
          text: 'RUN',
        };
      case 'SLEEP':
        return {
          icon: <RiPauseCircleLine className="h-3 w-3" />,
          kind: 'default' as const,
          text: 'SLEEP',
        };
      case 'WAIT_FOR_EVENT':
        return {
          icon: <RiMailLine className="h-3 w-3" />,
          kind: 'warning' as const,
          text: 'WAIT_FOR_EVENT',
        };
      case 'AI_GATEWAY':
        return {
          icon: <RiRobot2Fill className="h-3 w-3" />,
          kind: 'primary' as const,
          text: 'AI_GATEWAY',
        };
      case 'WAIT_FOR_SIGNAL':
        return {
          icon: <RiRadioButtonLine className="h-3 w-3" />,
          kind: 'warning' as const,
          text: 'WAIT_FOR_SIGNAL',
        };
      default:
        if (type.endsWith('sendSignal')) {
          return {
            icon: <RiSendPlaneFill className="h-3 w-3" />,
            kind: 'info' as const,
            text: type,
          };
        }
        if (type.endsWith('ai.infer')) {
          return {
            icon: <RiCpuFill className="h-3 w-3" />,
            kind: 'primary' as const,
            text: type,
          };
        }
        if (type.endsWith('ai.wrap')) {
          return {
            icon: <RiCpuFill className="h-3 w-3" />,
            kind: 'warning' as const,
            text: type,
          };
        }
        return {
          icon: <RiPlayFill className="h-3 w-3" />,
          kind: 'default' as const,
          text: type,
        };
    }
  };

  const config = getStepConfig(stepType);

  return <Pill kind={config.kind} icon={config.icon} iconSide="left" appearance="solidBright" />;
}
