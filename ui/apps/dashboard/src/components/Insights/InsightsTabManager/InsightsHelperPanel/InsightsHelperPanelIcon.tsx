'use client';

import { RiBookOpenLine, RiFeedbackLine, RiSparkling2Line, RiTable2 } from '@remixicon/react';

import {
  DOCUMENTATION,
  INSIGHTS_AI,
  SCHEMA_EXPLORER,
  SUPPORT,
  type HelperTitle,
} from './constants';

type InsightsHelperPanelIconProps = {
  className?: string;
  title: HelperTitle;
  size?: number;
};

export function InsightsHelperPanelIcon({
  className,
  title,
  size = 20,
}: InsightsHelperPanelIconProps) {
  switch (title) {
    case INSIGHTS_AI:
      return <RiSparkling2Line className={className} size={size} />;
    case DOCUMENTATION:
      return <RiBookOpenLine className={className} size={size} />;
    case SCHEMA_EXPLORER:
      return <RiTable2 className={className} size={size} />;
    case SUPPORT:
      return <RiFeedbackLine className={className} size={size} />;
    default:
      return null;
  }
}
