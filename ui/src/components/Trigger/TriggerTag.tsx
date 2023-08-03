import Tag from '@/components/Tag';
import { IconClock, IconEvent } from '@/icons';

type TriggerTagProps = {
  value: string;
  type: string;
};

export default function TriggerTag({ value, type }: TriggerTagProps) {
  return (
    <Tag>
      <div className="flex items-center gap-2">
        {type === 'EVENT' && <IconEvent className="h-2" />}
        {type === 'CRON' && <IconClock className="h-4" />}
        {value}
      </div>
    </Tag>
  );
}
