import { cn } from '@inngest/components/utils/classNames';

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
  const groupStyles = cn('flex items-center gap-1 rounded-md bg-canvasSubtle p-1', className);

  return (
    <div className={groupStyles} role="radiogroup" aria-label={title}>
      {options.map((option) => {
        const isSelected = option.id === selectedOption;
        const classNames = cn(
          'text-subtle hover:bg-canvasMuted border border-transparent hover:text-success font-medium px-3 py-1 rounded text-sm cursor-pointer',
          isSelected && 'bg-canvasBase border-muted text-basis cursor-default hover:bg-canvasBase'
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
