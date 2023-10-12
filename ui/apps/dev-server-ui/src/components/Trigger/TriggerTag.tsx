import Tag from '@/components/Tag';
import { IconClock, IconEvent } from '@/icons';
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
