"use client";

import {
  RiBookOpenLine,
  RiFeedbackLine,
  RiNodeTree,
  RiSparkling2Line,
} from "@remixicon/react";

import {
  DOCUMENTATION,
  INSIGHTS_AI,
  SCHEMA_EXPLORER,
  SUPPORT,
  type HelperTitle,
} from "./constants";

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
      return <RiSparkling2Line className={className} size={size} />;
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
