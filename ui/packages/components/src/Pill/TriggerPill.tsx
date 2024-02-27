import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';

import { type Trigger } from '../types/trigger';
import { Pill } from './Pill';

export function TriggerPill({ value, type }: Trigger) {
  return (
    <Pill>
      <div className="flex items-center gap-2">
        {type === 'EVENT' && <IconEvent className="text-indigo-500 dark:text-slate-400" />}
        {type === 'CRON' && <IconClock className="text-indigo-500 dark:text-slate-400" />}
        {value}
      </div>
    </Pill>
  );
}
