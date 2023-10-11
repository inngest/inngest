import { Time } from '@/components/Time';
import cn from '@/utils/cn';

const variantStyles = {
  dark: 'bg-slate-800 text-white',
  light: 'bg-slate-100 text-gray-900',
};

export default function DateCard({
  className,
  variant = 'light',
  date = '-',
  description,
}: {
  className?: string;
  variant?: 'dark' | 'light';
  date?: string;
  description: string;
}) {
  const classNames = cn(
    'py-2 pl-4 pr-12 rounded-lg overflow-hidden',
    variantStyles[variant],
    className
  );

  let time;
  if (date === '-') {
    time = <div className="text-sm">-</div>;
  } else {
    const dateObj = new Date(date);
    time = <Time className="text-sm" value={dateObj} />;
  }

  return (
    <div className={classNames}>
      {time}
      <div className="text-xs text-slate-400">{description}</div>
    </div>
  );
}
