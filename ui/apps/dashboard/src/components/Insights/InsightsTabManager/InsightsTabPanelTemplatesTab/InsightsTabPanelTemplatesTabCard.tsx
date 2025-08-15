'use client';

import { RiAlertLine, RiErrorWarningLine, RiTimeLine } from '@remixicon/react';

import type { QueryTemplate } from '@/components/Insights/types';
import { useTabManagerActions } from '../TabManagerContext';

const APPEARANCE_CLASSES = 'bg-canvasBase border-subtle rounded-[4px] border';
const CONTENT_LAYOUT_CLASSES = 'flex flex-col gap-1';
const DESCRIPTION_CLASSES = 'text-muted text-sm';
const INTERACTION_CLASSES = 'hover:bg-canvasSubtle transition-colors';
const LAYOUT_CLASSES = 'flex flex-col gap-3 h-[152px] items-start w-[256px]';
const SHADOW_CLASSES =
  'shadow-[0_1px_1px_-0.5px_rgba(42,51,69,0.04),0_6px_6px_-3px_rgba(42,51,69,0.04)]';
const SPACING_CLASSES = 'p-4';
const TEXT_CLASSES = 'text-left';
const TITLE_CLASSES = 'text-basis';

const BUTTON_CARD_STYLES = `${LAYOUT_CLASSES} ${APPEARANCE_CLASSES} ${INTERACTION_CLASSES} ${SPACING_CLASSES} ${TEXT_CLASSES} ${SHADOW_CLASSES}`;

const TEMPLATE_KIND_CONFIG: Record<
  QueryTemplate['templateKind'],
  {
    backgroundColor: string;
    icon: React.ComponentType<{ className?: string }>;
    textColor: string;
  }
> = {
  error: {
    backgroundColor: 'bg-error',
    icon: RiErrorWarningLine,
    textColor: 'text-error',
  },
  time: {
    backgroundColor: 'bg-info',
    icon: RiTimeLine,
    textColor: 'text-info',
  },
  warning: {
    backgroundColor: 'bg-warning',
    icon: RiAlertLine,
    textColor: 'text-warning',
  },
};

interface InsightsTabPanelTemplatesTabCardProps {
  template: QueryTemplate;
}

export function InsightsTabPanelTemplatesTabCard({
  template,
}: InsightsTabPanelTemplatesTabCardProps) {
  const { tabManagerActions } = useTabManagerActions();
  const config = TEMPLATE_KIND_CONFIG[template.templateKind];
  const IconComponent = config.icon;

  return (
    <button
      className={BUTTON_CARD_STYLES}
      onClick={() => {
        tabManagerActions.createTabFromTemplate(template);
      }}
    >
      <div
        className={`${config.backgroundColor} flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md`}
      >
        <IconComponent className={`${config.textColor} h-5 w-5 flex-shrink-0`} />
      </div>
      <div className={CONTENT_LAYOUT_CLASSES}>
        <h3 className={TITLE_CLASSES}>{template.name}</h3>
        <p className={DESCRIPTION_CLASSES}>{template.explanation}</p>
      </div>
    </button>
  );
}
