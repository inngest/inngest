import Tag from '@/components/Tag';
import { IconEvent, IconClock } from '@/icons';

type TriggerTagProps = {
  name: string;
  type: string;
};

export default function TriggerTag({ name, type }: TriggerTagProps) {
  return (
    <Tag>
      <div className="flex items-center gap-2">
        {type === 'EVENT' && <IconEvent className="h-2" />}
        {type === 'CRON' && <IconClock className="h-4" />}
        {name}
      </div>
    </Tag>
  );
}
