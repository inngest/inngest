import { IconMagnifyingGlass } from '@/icons';
import classNames from '@/utils/classnames';

interface SearchInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  value: string;
  className?: string;
  onChange: (e) => void;
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
        'relative text-slate-400 flex items-center bg-slate-950 pl-6 ',
        className,
      )}
    >
      <input
        type="text"
        className="text-slate-100 w-96 placeholder-slate-400 py-1 pl-4 bg-transparent"
        placeholder={props?.placeholder ?? 'Search...'}
        value={value ?? ''}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
          onChange(e.target.value);
          debouncedSearch();
        }}
        {...props}
      />
      <IconMagnifyingGlass className="absolute left-6 h-3 w-3 text--slate-400" />
    </div>
  );
}
