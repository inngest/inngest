import { type UrlObject } from 'url';
import type { Route } from 'next';
import { ClockIcon } from '@heroicons/react/20/solid';

import { Pill } from '@/components/Pill/Pill';
import EventIcon from '@/icons/event.svg';

export const TRIGGER_TYPE = {
  event: 'event',
  schedule: 'schedule',
} as const;

export type TriggerType = (typeof TRIGGER_TYPE)[keyof typeof TRIGGER_TYPE];

export type Trigger = {
  type: TriggerType;
  value: string;
};

export const triggerIcons = {
  event: EventIcon,
  schedule: ClockIcon,
} as const;

type TriggerPillProps<PassedHref extends string> = {
  href?: Route<PassedHref> | UrlObject;
  trigger: Trigger;
};

export default function TriggerPill<PassedHref extends string>({
  href,
  trigger,
}: TriggerPillProps<PassedHref>) {
  const Icon = triggerIcons[trigger.type];

  return (
    <Pill href={href} className="bg-white align-middle text-slate-600" key={trigger.value}>
      <Icon className="mr-1 h-3.5 w-[18px] text-indigo-500" />
      {trigger.value}
    </Pill>
  );
}
