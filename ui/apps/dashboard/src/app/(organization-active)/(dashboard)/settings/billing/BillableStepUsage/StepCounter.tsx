import cn from '@/utils/cn';

type Props = {
  count: number | undefined;
  numberClassName?: string;
  title: string;
};

export function StepCounter({ count, numberClassName, title }: Props) {
  let text: string;
  if (count !== undefined) {
    text = count.toLocaleString();
  } else {
    text = 'Unknown';
  }

  return (
    <div className="text-right">
      <h2 className={cn('text-[1.375rem] font-semibold', numberClassName)}>{text}</h2>
      <div className="text-sm font-medium text-slate-500">{title}</div>
    </div>
  );
}
