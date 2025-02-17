import { cn } from '@inngest/components/utils/classNames';
import { RiSearchLine } from '@remixicon/react';

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
      className={cn(
        'bg-canvasBase text-muted border-subtle relative flex items-center border pl-4 text-sm',
        className
      )}
    >
      <input
        type="text"
        className="text-muted placeholder-subtle w-96 bg-transparent py-1 pl-4 outline-none"
        placeholder={props?.placeholder ?? 'Search...'}
        value={value ?? ''}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
          onChange(e.target.value);
          debouncedSearch();
        }}
        {...props}
      />
      <RiSearchLine className="text-muted absolute left-2 h-4 w-4" />
    </div>
  );
}
