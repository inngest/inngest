import { IconMagnifyingGlass } from '@/icons';
import classNames from '@/utils/classnames';

interface SearchInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange'> {
  value: string;
  className?: string;
  onChange: (value: string) => void;
  debouncedSearch: () => void;
}

export default function SearchInput({
  value,
  onChange,
  debouncedSearch,
  className,
  ...props
}: SearchInputProps) {
  return (
    <div
      className={classNames(
        'bg-slate-910 relative flex items-center pl-6 text-slate-400 ',
        className ?? ''
      )}
    >
      <input
        type="text"
        className="w-96 bg-transparent py-1 pl-4 text-slate-100 placeholder-slate-400"
        placeholder={props?.placeholder ?? 'Search...'}
        value={value ?? ''}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
          onChange(e.target.value);
          debouncedSearch();
        }}
        {...props}
      />
      <IconMagnifyingGlass className="text--slate-400 absolute left-6 h-3 w-3" />
    </div>
  );
}
