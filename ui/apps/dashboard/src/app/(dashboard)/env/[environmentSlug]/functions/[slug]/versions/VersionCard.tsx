import Block from '@/components/Block';
import cn from '@/utils/cn';

const statusStyles = {
  highlighted: 'bg-slate-900 text-white py-6',
  disabled: 'bg-white text-gray-900',
};

export default function VersionCard({
  icon,
  name,
  badge,
  dateCards,
  button,
  className,
  status = 'highlighted',
}: {
  icon: React.ReactNode;
  name: number;
  badge?: React.ReactNode;
  dateCards?: React.ReactNode;
  button?: React.ReactNode;
  className?: string;
  status?: 'highlighted' | 'disabled';
}) {
  const classNames = cn(
    'rounded-lg max-w-5xl m-auto px-2 py-4 mb-2 drop-shadow text-sm',
    statusStyles[status],
    className
  );

  return (
    <li>
      <Block className={classNames}>
        <div className="flex items-center">
          <div className="p-4 pr-2">{icon}</div>
          <div className="flex-1">
            version {name}
            {badge}
          </div>
          {dateCards}
          <div className="pr-2">{button}</div>
        </div>
      </Block>
    </li>
  );
}
