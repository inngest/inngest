import cn from '@/utils/cn';

type Options = Readonly<
  {
    name: string;
    id: string;
    icon?: React.ReactNode;
  }[]
>;

type GroupButtonProps<T extends Options> = {
  title: string;
  selectedOption: string;
  handleClick: (id: T[number]['id']) => void;
  options: T;
  className?: string;
};

export default function GroupButton<T extends Options>({
  title,
  options,
  handleClick,
  selectedOption,
  className,
}: GroupButtonProps<T>) {
  const groupStyles = cn('flex items-center gap-1 rounded-lg bg-slate-50 p-1', className);

  return (
    <div className={groupStyles} role="radiogroup" aria-label={title}>
      {options?.map((option) => {
        const isSelected = option.id === selectedOption;
        const classNames = cn(
          'text-slate-400 hover:bg-slate-100 hover:text-indigo-500 font-medium px-3 py-1 rounded-sm text-sm cursor-pointer',
          isSelected &&
            'bg-white shadow-outline-secondary-light text-slate-700 cursor-default hover:bg-white hover:text-slate-700'
        );

        return (
          <button
            key={option.id}
            id={option.id}
            role="radio"
            className={classNames}
            onClick={() => handleClick(option.id)}
            disabled={isSelected}
            aria-checked={isSelected}
          >
            {option.icon}
            {option.name}
          </button>
        );
      })}
    </div>
  );
}
