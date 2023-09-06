import Tooltip from '@/components/Tooltip/Tooltip';
import { IconInfo } from '@/icons';
import classNames from '@/utils/classnames';

export type MetadataItemProps = {
  label: String;
  value: String;
  tooltip?: String;
  size?: 'small' | 'large';
};

export default function MetadataItem({ value, label, tooltip }: MetadataItemProps) {
  return (
    <div className={classNames('flex flex-col p-1.5 bg-slate-950')}>
      <span className="text-white">{value}</span>
      <span className="flex items-center gap-1">
        <span className="text-slate-500 capitalize">{label}</span>
        {tooltip && (
          <Tooltip children={<IconInfo className="text-slate-400 icon-lg" />} content={tooltip} />
        )}
      </span>
    </div>
  );
}
