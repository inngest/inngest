import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';

import Tag from '@/components/Tag';
import { FunctionTriggerTypes } from '@/store/generated';

type TriggerTagProps = {
  value: string;
  type: string;
};

export default function TriggerTag({ value, type }: TriggerTagProps) {
  return (
    <Tag>
      <div className="flex items-center gap-2 text-slate-400">
        {type === FunctionTriggerTypes.Event && <IconEvent />}
        {type === FunctionTriggerTypes.Cron && <IconClock />}
        {value}
      </div>
    </Tag>
  );
}
