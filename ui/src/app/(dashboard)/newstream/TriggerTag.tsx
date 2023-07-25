import Tag from '@/components/Tag';
import { IconEvent, IconClock } from '@/icons';

export default function TriggerTag({ name, type }) {
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
