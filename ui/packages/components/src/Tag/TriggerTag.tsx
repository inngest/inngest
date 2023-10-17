import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';
import { TriggerTypes } from '@inngest/components/types/triggers';

import { Tag } from './Tag';

type TriggerTagProps = {
  value: string;
  type: TriggerTypes;
};

export function TriggerTag({ value, type }: TriggerTagProps) {
  return (
    <Tag>
      <div className="flex items-center gap-2 text-slate-400">
        {type === TriggerTypes.Event && <IconEvent />}
        {type === TriggerTypes.Cron && <IconClock />}
        {value}
      </div>
    </Tag>
  );
}
