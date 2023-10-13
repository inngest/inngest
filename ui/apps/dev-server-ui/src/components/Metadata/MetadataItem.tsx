import { Tooltip } from '@inngest/components/Tooltip/Tooltip';
import { classNames } from '@inngest/components/utils/classNames';

import { IconInfo } from '@/icons';

export type MetadataItemProps = {
  className?: string;
  label: string;
  value: string | JSX.Element;
  title?: string;
  tooltip?: string;
  type?: 'code' | 'text';
  size?: 'small' | 'large';
};

export default function MetadataItem({
  className,
  value,
  title,
  label,
  type,
  tooltip,
}: MetadataItemProps) {
  return (
    <div className={classNames('flex flex-col p-1.5', className)}>
      <span
        title={title}
        className={classNames(type === 'code' && 'font-mono', 'text-sm text-white')}
      >
        {value}
      </span>
      <span className="flex items-center gap-1">
        <span className="text-sm capitalize text-slate-500">{label}</span>
        {tooltip && (
          <Tooltip children={<IconInfo className="h-4 w-4 text-slate-400" />} content={tooltip} />
        )}
      </span>
    </div>
  );
}
