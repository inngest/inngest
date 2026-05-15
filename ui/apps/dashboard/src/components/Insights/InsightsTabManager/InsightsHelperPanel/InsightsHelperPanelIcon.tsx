import {
  RiBookOpenLine,
  RiFeedbackLine,
  RiNodeTree,
  RiTableLine,
} from '@remixicon/react';

import InsightsAI from '@/components/Icons/insights-ai-icon.svg?react';
import InsightsAIDark from '@/components/Icons/insights-ai-icon-dark.svg?react';

import {
  CELL_DETAIL,
  DOCUMENTATION,
  INSIGHTS_AI,
  SCHEMA_EXPLORER,
  SUPPORT,
  type HelperTitle,
} from './constants';

type InsightsHelperPanelIconProps = {
  className?: string;
  size?: number;
  title: HelperTitle;
};

export function InsightsHelperPanelIcon({
  className,
  size = 20,
  title,
}: InsightsHelperPanelIconProps) {
  switch (title) {
    case INSIGHTS_AI:
      return (
        <>
          <InsightsAI
            className={`${className} block dark:hidden`}
            width={size}
            height={size}
          />
          <InsightsAIDark
            className={`${className} hidden dark:block`}
            width={size}
            height={size}
          />
        </>
      );
    case CELL_DETAIL:
      return <RiTableLine className={className} size={size} />;
    case DOCUMENTATION:
      return <RiBookOpenLine className={className} size={size} />;
    case SCHEMA_EXPLORER:
      return <RiNodeTree className={className} size={size} />;
    case SUPPORT:
      return <RiFeedbackLine className={className} size={size} />;
    default:
      return null;
  }
}
