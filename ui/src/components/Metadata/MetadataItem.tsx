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
      <span className="text-slate-500 capitalize">{label}</span>
    </div>
  );
}
